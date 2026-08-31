package toudocu

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func newReviewRepository(t *testing.T) (string, string) {
	t.Helper()
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(root, "server.go"), "package main\n\nfunc hello() {}\n")
	gitTestRun(t, root, "add", "server.go")
	gitTestRun(t, root, "commit", "-q", "-m", "review baseline")
	return root, docs
}

func reviewOptions(root, docs string) Options {
	return Options{RepositoryRoot: root, InputDirectory: docs, ChangeBase: "HEAD", ChangeTarget: "working-tree", ChangeRenameSimilarity: 60}
}

func guard(state ReviewState) ReviewMutationGuard {
	return ReviewMutationGuard{ExpectedRevision: state.Revision, ExpectedStateDigest: state.StateDigest}
}

func TestRepositoryReviewProjectionAndFile(t *testing.T) {
	root, docs := newReviewRepository(t)
	writeChangesTestFile(t, filepath.Join(root, "server.go"), "package main\n\nfunc changed() {}\n")
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\n- Status: Active\n\n## Rules\n\nUpdated.\n")
	report, err := BuildRepositoryReview(reviewOptions(root, docs))
	if err != nil || !report.FeedbackWritable || len(report.Files) != 2 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	detail, err := BuildRepositoryReviewFile(reviewOptions(root, docs), "server.go")
	if err != nil || detail.Current == nil || detail.Before == nil || len(detail.Hunks) == 0 {
		t.Fatalf("detail=%#v err=%v", detail, err)
	}
	if _, err := BuildRepositoryReviewFile(reviewOptions(root, docs), "../.git/config"); reviewErrorCode(err) != "REVIEW_UNSAFE_PATH" {
		t.Fatalf("unsafe path error=%v", err)
	}
}

func TestRepositoryReviewFileIncludesSafeRenderedMarkdown(t *testing.T) {
	root, docs := newReviewRepository(t)
	path := filepath.Join(docs, "modules", "MOD-CORE.md")
	writeChangesTestFile(t, path, "# MOD-CORE: Core\n\n<script>alert(1)</script>\n\n## Rules\n\nRendered current.\n")
	server := &documentationServer{options: reviewOptions(root, docs), changesCache: map[string]*ChangeSetReport{}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, reviewRepositoryAPIBase+"/repository/file?path=docs/modules/MOD-CORE.md", nil)
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", response.Code, response.Body.String())
	}
	var detail RepositoryReviewFileDetail
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.RenderedBefore == nil || detail.RenderedCurrent == nil || !strings.Contains(*detail.RenderedCurrent, "Rendered current.") {
		t.Fatalf("rendered detail=%#v", detail)
	}
	if strings.Contains(strings.ToLower(*detail.RenderedCurrent), "<script") {
		t.Fatalf("unsafe rendered detail: %s", *detail.RenderedCurrent)
	}
}

func TestAgentFeedbackLifecycle(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.discussions()
	if err != nil || state.Session != nil || state.StateDigest == "" {
		t.Fatalf("initial=%#v err=%v", state, err)
	}
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"},
		Selection: &SelectionHint{SelectedText: "Rules"}, Intent: "question", Text: "Почему это правило нужно?",
	})
	if err != nil || state.Session == nil || len(state.Session.Discussions) != 1 {
		t.Fatalf("create=%#v err=%v", state, err)
	}
	discussion := state.Session.Discussions[0]
	message := discussion.Messages[0]
	if message.State != "submitted" || len(state.Deliveries) != 1 || state.Deliveries[0].State != "pending" || discussion.Anchor == nil || discussion.Anchor.SelectedText != "Rules" || discussion.Target.Range == nil {
		t.Fatalf("discussion=%#v", discussion)
	}
	state, err = service.updateMessage(discussion.ID, message.ID, UpdateMessageRequest{ReviewMutationGuard: guard(state), Intent: "question", Text: "Объясни правило."})
	if err != nil || state.Session.Discussions[0].Messages[0].Text != "Объясни правило." {
		t.Fatalf("edit=%#v err=%v", state, err)
	}
	delivery := state.Deliveries[0]
	request, err := service.claimNext()
	if err != nil || !request.Pending || request.DeliveryID != delivery.ID || request.Target.AnchorState != "current" || request.PendingCount != 1 {
		t.Fatalf("next=%#v err=%v", request, err)
	}
	state, err = service.discussions()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.updateMessage(discussion.ID, message.ID, UpdateMessageRequest{ReviewMutationGuard: guard(state), Intent: "question", Text: "late edit"}); reviewErrorCode(err) != "AGENT_INVALID_MESSAGE" {
		t.Fatalf("claimed message changed: %v", err)
	}
	if _, err := service.claimNext(); reviewErrorCode(err) != "AGENT_INBOX_BUSY" {
		t.Fatalf("second consumer error=%v", err)
	}
	invalid := AgentResponse{SchemaVersion: 1, DeliveryID: delivery.ID, DiscussionID: discussion.ID, Outcome: "changed", Message: "changed", ChangedPaths: []string{"docs/modules/MOD-CORE.md"}}
	if _, err := service.respond(invalid); reviewErrorCode(err) != "AGENT_INVALID_MESSAGE" {
		t.Fatalf("question changed error=%v", err)
	}
	response := AgentResponse{SchemaVersion: 1, DeliveryID: delivery.ID, DiscussionID: discussion.ID, Outcome: "answered", Message: "Правило ограничивает область.", Evidence: []AgentEvidence{{Path: "server.go", StartLine: 1, EndLine: 3}}}
	state, err = service.respond(response)
	if err != nil || state.Session.Discussions[0].State != "open" || len(state.Session.Discussions[0].Messages) != 2 || state.Deliveries[0].State != "responded" {
		t.Fatalf("respond=%#v err=%v", state, err)
	}
	revision := state.Revision
	state, err = service.respond(response)
	if err != nil || state.Revision != revision {
		t.Fatalf("idempotent response=%#v err=%v", state, err)
	}
	response.Message = "different"
	if _, err := service.respond(response); reviewErrorCode(err) != "AGENT_RESPONSE_CONFLICT" {
		t.Fatalf("response conflict=%v", err)
	}

	state, err = service.createMessage(discussion.ID, CreateMessageRequest{ReviewMutationGuard: guard(state), Intent: "change_request", Text: "Тогда уточни раздел."})
	if err != nil {
		t.Fatal(err)
	}
	second := state.Deliveries[1]
	if second.Sequence != 2 {
		t.Fatalf("follow-up=%#v", second)
	}
	request, err = service.claimNext()
	if err != nil || request.DeliveryID != second.ID || len(request.Discussion.Messages) != 3 {
		t.Fatalf("follow-up request=%#v err=%v", request, err)
	}
	state, err = service.respond(AgentResponse{SchemaVersion: 1, DeliveryID: second.ID, DiscussionID: discussion.ID, Outcome: "changed", Message: "Раздел уточнён.", ChangedPaths: []string{"docs/modules/MOD-CORE.md"}})
	if err != nil || len(state.Session.Discussions[0].Messages) != 4 {
		t.Fatalf("follow-up response=%#v err=%v", state, err)
	}
	state, err = service.updateDiscussion(discussion.ID, UpdateDiscussionRequest{ReviewMutationGuard: guard(state), State: "resolved"})
	if err != nil || state.Session.Discussions[0].State != "resolved" {
		t.Fatalf("resolve=%#v err=%v", state, err)
	}
	state, err = service.deleteDiscussion(discussion.ID, guard(state))
	if err != nil || len(state.Session.Discussions) != 0 || len(state.Deliveries) != 0 {
		t.Fatalf("delete=%#v err=%v", state, err)
	}
}

func TestAgentFeedbackAcceptsDraftDocument(t *testing.T) {
	root, docs := newReviewRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "drafts", "entry.md"), "# Draft\n\nText.\n")
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.discussions()
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/drafts/entry.md"},
		Intent: "question", Text: "Нужно уточнение.",
	})
	if err != nil || len(state.Session.Discussions) != 1 || state.Session.Discussions[0].Target.Path != "docs/drafts/entry.md" {
		t.Fatalf("draft discussion: %#v %v", state, err)
	}
}

func TestAgentFeedbackFileTargets(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	writeChangesTestFile(t, filepath.Join(root, "server.go"), "package main\n\nfunc changed() { println(\"привет\") }\n")
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.discussions()
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "file", Path: "server.go", Range: &ReviewRange{Start: ReviewPosition{Line: 3, Column: 6}, End: ReviewPosition{Line: 3, Column: 13}}},
		Intent: "change_request", Text: "Переименуй функцию.",
	})
	if err != nil || state.Session.Discussions[0].Anchor.SelectedText != "changed" {
		t.Fatalf("file range=%#v err=%v", state, err)
	}
	next, err := service.claimNext()
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.respond(AgentResponse{SchemaVersion: 1, DeliveryID: next.DeliveryID, DiscussionID: next.Discussion.ID, Outcome: "changed", Message: "Готово.", ChangedPaths: []string{"server.go"}})
	if err != nil || state.Deliveries[0].State != "responded" {
		t.Fatalf("file response=%#v err=%v", state, err)
	}
	if _, err := service.createDiscussion(CreateDiscussionRequest{ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "file", Path: "README.md"}, Intent: "question", Text: "Почему?"}); reviewErrorCode(err) != "AGENT_INVALID_TARGET" {
		t.Fatalf("unchanged file accepted: %v", err)
	}

	writeChangesTestFile(t, filepath.Join(root, "binary.dat"), "baseline\n")
	writeChangesTestFile(t, filepath.Join(root, "large.txt"), "baseline\n")
	gitTestRun(t, root, "add", "binary.dat", "large.txt")
	gitTestRun(t, root, "commit", "-q", "-m", "binary baseline")
	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}
	state, _ = service.discussions()
	state, err = service.createDiscussion(CreateDiscussionRequest{ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "file", Path: "binary.dat"}, Intent: "question", Text: "Что изменилось?"})
	if err != nil || state.Session.Discussions[len(state.Session.Discussions)-1].Target.Range != nil {
		t.Fatalf("binary whole-file target=%#v err=%v", state, err)
	}
	if _, err := service.createDiscussion(CreateDiscussionRequest{ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "file", Path: "binary.dat", Range: &ReviewRange{Start: ReviewPosition{Line: 1, Column: 1}, End: ReviewPosition{Line: 1, Column: 2}}}, Intent: "question", Text: "Где?"}); err == nil {
		t.Fatalf("binary range accepted: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.txt"), bytes.Repeat([]byte{'x'}, reviewSnapshotLimit+1), 0o600); err != nil {
		t.Fatal(err)
	}
	state, _ = service.discussions()
	if _, err := service.createDiscussion(CreateDiscussionRequest{ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "file", Path: "large.txt"}, Intent: "question", Text: "Почему файл большой?"}); err != nil {
		t.Fatalf("large whole-file target: %v", err)
	}
}

func TestAgentFeedbackDeletedFileCanStartAndContinueDiscussion(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	if err := os.Remove(filepath.Join(root, "server.go")); err != nil {
		t.Fatal(err)
	}
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.discussions()
	state, err = service.createDiscussion(CreateDiscussionRequest{ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "file", Path: "server.go"}, Intent: "question", Text: "Почему удалён?"})
	if err != nil || state.Session.Discussions[0].Placement.Status != "deleted" {
		t.Fatalf("deleted create=%#v err=%v", state, err)
	}
	next, err := service.claimNext()
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.respond(AgentResponse{SchemaVersion: 1, DeliveryID: next.DeliveryID, DiscussionID: next.Discussion.ID, Outcome: "answered", Message: "Файл удалён в diff."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.createMessage(next.Discussion.ID, CreateMessageRequest{ReviewMutationGuard: guard(state), Intent: "question", Text: "Это намеренно?"}); err != nil {
		t.Fatalf("deleted follow-up: %v", err)
	}
}

func TestAgentFeedbackSelectsRepeatedTextByOccurrence(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	path := filepath.Join(docs, "modules", "MOD-CORE.md")
	writeChangesTestFile(t, path, "# Core\n\nRepeated\ntext.\n\nRepeated text.\n")
	secondLine := fileLineContaining(t, path, "Repeated text.")
	firstLine := fileLineContaining(t, path, "Repeated")
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.discussions()
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"},
		Selection: &SelectionHint{SelectedText: "Repeated text.", Occurrence: 2}, Intent: "question", Text: "Почему повтор?",
	})
	if err != nil || state.Session.Discussions[0].Target.Range.Start.Line != secondLine {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"},
		Selection: &SelectionHint{SelectedText: "Repeated text."}, Intent: "question", Text: "Почему перенос?",
	})
	if err != nil || state.Session.Discussions[1].Target.Range.Start.Line != firstLine || state.Session.Discussions[1].Anchor.SelectedText != "Repeated\ntext." {
		t.Fatalf("state=%#v err=%v", state, err)
	}
}

func TestAgentFeedbackKeepsUnmatchedDocumentSelection(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.discussions()
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"},
		Selection: &SelectionHint{SelectedText: "Rendered Mermaid label"}, Intent: "question", Text: "Что означает подпись?",
	})
	if err != nil {
		t.Fatal(err)
	}
	discussion := state.Session.Discussions[0]
	if discussion.Target.Range != nil || discussion.Anchor.Range != nil || discussion.Anchor.SelectedText != "Rendered Mermaid label" {
		t.Fatalf("discussion=%#v", discussion)
	}
	next, err := service.claimNext()
	if err != nil || next.Target.Path != "docs/modules/MOD-CORE.md" || next.Target.Range != nil || next.Target.SelectedText != "Rendered Mermaid label" {
		t.Fatalf("next=%#v err=%v", next, err)
	}
	state, err = service.respond(AgentResponse{SchemaVersion: 1, DeliveryID: next.DeliveryID, DiscussionID: next.Discussion.ID, Outcome: "answered", Message: "Это подпись шага."})
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.createMessage(discussion.ID, CreateMessageRequest{ReviewMutationGuard: guard(state), Intent: "question", Text: "А подробнее?"})
	if err != nil || state.Session.Discussions[0].Anchor.SelectedText != "Rendered Mermaid label" {
		t.Fatalf("follow-up=%#v err=%v", state, err)
	}
	anchor, target, err := service.captureAnchor(ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"}, &SelectionHint{SelectedText: "Original.", Occurrence: 7})
	if err != nil || target.Range != nil || anchor.Range != nil || anchor.SelectedText != "Original." {
		t.Fatalf("invalid occurrence anchor=%#v target=%#v err=%v", anchor, target, err)
	}
}

func TestAgentFeedbackPersistsUTF8AnchorContext(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	selected := "выделение"
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), strings.Repeat("界", 800)+selected+strings.Repeat("界", 800))
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.discussions()
	state, err = service.createDiscussion(CreateDiscussionRequest{
		ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"},
		Selection: &SelectionHint{SelectedText: selected}, Intent: "question", Text: "Почему этот фрагмент?",
	})
	if err != nil {
		t.Fatal(err)
	}
	anchor := state.Session.Discussions[0].Anchor
	if !utf8.ValidString(anchor.ContextBefore) || !utf8.ValidString(anchor.ContextAfter) || anchor.SelectedText != selected {
		t.Fatalf("anchor=%#v", anchor)
	}
	restarted, err := newReviewService(Options{RepositoryRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := restarted.discussions()
	if err != nil || reloaded.Session.Discussions[0].Anchor.SelectedText != selected {
		t.Fatalf("restart=%#v err=%v", reloaded, err)
	}
}

func TestAgentFeedbackFIFOReanchorPersistenceAndConcurrency(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	service, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.discussions()
	createAndSubmit := func(text string) *AgentDelivery {
		var createErr error
		state, createErr = service.createDiscussion(CreateDiscussionRequest{ReviewMutationGuard: guard(state), Target: ReviewTarget{Kind: "document", Path: "docs/modules/MOD-CORE.md"}, Selection: &SelectionHint{SelectedText: "Original."}, Intent: "question", Text: text})
		if createErr != nil {
			t.Fatal(createErr)
		}
		delivery := state.Deliveries[len(state.Deliveries)-1]
		return &delivery
	}
	first := createAndSubmit("first")
	second := createAndSubmit("second")
	next, err := service.claimNext()
	if err != nil || next.DeliveryID != first.ID || next.PendingCount != 2 || !next.HasMore {
		t.Fatalf("fifo next=%#v err=%v second=%s", next, err, second.ID)
	}
	service.now = func() time.Time { return time.Now().UTC().Add(reviewClaimLease + time.Minute) }
	retry, err := service.claimNext()
	if err != nil || retry.DeliveryID != first.ID {
		t.Fatalf("retry=%#v err=%v", retry, err)
	}

	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nInserted.\n\n- Status: Active\n\n## Rules\n\nOriginal.\n")
	originalLine := fileLineContaining(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "Original.")
	loaded, err := service.discussions()
	if err != nil || loaded.Session.Discussions[0].Placement.Status != "moved" || loaded.Session.Discussions[0].Placement.Range.Start.Line != originalLine {
		t.Fatalf("moved=%#v err=%v", loaded.Session.Discussions[0].Placement, err)
	}
	restarted, err := newReviewService(Options{RepositoryRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := restarted.discussions()
	if err != nil || reloaded.DocsPath != "docs" || len(reloaded.Deliveries) != 2 {
		t.Fatalf("restart=%#v err=%v", reloaded, err)
	}

	parallel, err := newReviewService(reviewOptions(root, docs))
	if err != nil {
		t.Fatal(err)
	}
	current, _ := parallel.discussions()
	request := CreateDiscussionRequest{ReviewMutationGuard: guard(current), Target: ReviewTarget{Kind: "document", Path: "docs/use-cases/UC-CORE-01.md"}, Intent: "question", Text: "concurrent"}
	var wait sync.WaitGroup
	wait.Add(2)
	errors := make(chan error, 2)
	for range 2 {
		go func() { defer wait.Done(); _, createErr := parallel.createDiscussion(request); errors <- createErr }()
	}
	wait.Wait()
	close(errors)
	successes, rejections := 0, 0
	for createErr := range errors {
		if createErr == nil {
			successes++
		} else if code := reviewErrorCode(createErr); code == "AGENT_REVISION_CONFLICT" || code == "AGENT_INBOX_BUSY" {
			rejections++
		} else {
			t.Fatal(createErr)
		}
	}
	if successes != 1 || rejections != 1 {
		t.Fatalf("successes=%d rejections=%d", successes, rejections)
	}

	if err := os.WriteFile(service.store.statePath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.discussions(); reviewErrorCode(err) != "AGENT_STATE_CORRUPTED" {
		t.Fatalf("corruption error=%v", err)
	}
}

func TestAgentHTTPAndCLI(t *testing.T) {
	root, docs := newReviewRepository(t)
	t.Setenv("TOUDOCU_STATE_HOME", t.TempDir())
	server := &documentationServer{options: reviewOptions(root, docs), changesCache: map[string]*ChangeSetReport{}}
	initial := httptest.NewRecorder()
	server.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, reviewAPIBase+"/discussions", nil))
	if initial.Code != http.StatusOK {
		t.Fatalf("list: %d %s", initial.Code, initial.Body.String())
	}
	var state ReviewState
	if err := json.Unmarshal(initial.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	body := `{"expectedRevision":0,"expectedStateDigest":"` + state.StateDigest + `","target":{"kind":"document","path":"docs/modules/MOD-CORE.md"},"intent":"question","text":"Что делает модуль?"}`
	badBody := strings.TrimSuffix(body, "}") + `,"unknown":true}`
	badRequest := httptest.NewRequest(http.MethodPost, reviewAPIBase+"/discussions", strings.NewReader(badBody))
	badRequest.Header.Set("Content-Type", "application/json")
	badRequest.Header.Set("X-Toudocu-Action", "agent-discussion-create")
	badRequest.Header.Set("Origin", "http://example.com")
	badHTTP := httptest.NewRecorder()
	server.ServeHTTP(badHTTP, badRequest)
	if badHTTP.Code != http.StatusBadRequest {
		t.Fatalf("unknown property: %d %s", badHTTP.Code, badHTTP.Body.String())
	}
	wrongAction := httptest.NewRequest(http.MethodPost, reviewAPIBase+"/discussions", strings.NewReader(body))
	wrongAction.Header.Set("Content-Type", "application/json")
	wrongAction.Header.Set("X-Toudocu-Action", "agent-message-create")
	wrongAction.Header.Set("Origin", "http://example.com")
	wrongActionHTTP := httptest.NewRecorder()
	server.ServeHTTP(wrongActionHTTP, wrongAction)
	if wrongActionHTTP.Code != http.StatusForbidden {
		t.Fatalf("wrong action: %d %s", wrongActionHTTP.Code, wrongActionHTTP.Body.String())
	}
	missingOrigin := httptest.NewRequest(http.MethodPost, reviewAPIBase+"/discussions", strings.NewReader(body))
	missingOrigin.Header.Set("Content-Type", "application/json")
	missingOrigin.Header.Set("X-Toudocu-Action", "agent-discussion-create")
	missingOriginHTTP := httptest.NewRecorder()
	server.ServeHTTP(missingOriginHTTP, missingOrigin)
	if missingOriginHTTP.Code != http.StatusForbidden {
		t.Fatalf("missing same-origin proof: %d %s", missingOriginHTTP.Code, missingOriginHTTP.Body.String())
	}
	createRequest := httptest.NewRequest(http.MethodPost, reviewAPIBase+"/discussions", strings.NewReader(body))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("X-Toudocu-Action", "agent-discussion-create")
	createRequest.Header.Set("Origin", "http://example.com")
	createdHTTP := httptest.NewRecorder()
	server.ServeHTTP(createdHTTP, createRequest)
	if createdHTTP.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", createdHTTP.Code, createdHTTP.Body.String())
	}
	if err := json.Unmarshal(createdHTTP.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr strings.Builder
	if code := runAgentCLI([]string{"agent", "next", "--repository-root", root, "--json"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("next exit=%d stderr=%s", code, stderr.String())
	}
	var next AgentRequest
	if err := json.Unmarshal([]byte(stdout.String()), &next); err != nil || !next.Pending {
		t.Fatalf("next=%#v err=%v", next, err)
	}
	response := AgentResponse{SchemaVersion: 1, DeliveryID: next.DeliveryID, DiscussionID: next.Discussion.ID, Outcome: "answered", Message: "Это ядро."}
	responseJSON, _ := json.Marshal(response)
	stdout.Reset()
	stderr.Reset()
	if code := runAgentCLI([]string{"agent", "respond", "--repository-root", root, "--json"}, strings.NewReader(string(responseJSON)), &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), `"accepted": true`) {
		t.Fatalf("respond exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	if code := runAgentCLI([]string{"agent", "next", "--repository-root", root, "--json"}, strings.NewReader(""), &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), `"pending": false`) {
		t.Fatalf("empty next exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	repositoryHTTP := httptest.NewRecorder()
	server.ServeHTTP(repositoryHTTP, httptest.NewRequest(http.MethodGet, reviewRepositoryAPIBase+"/repository/changes", nil))
	if repositoryHTTP.Code != http.StatusOK || !strings.Contains(repositoryHTTP.Body.String(), `"feedbackWritable":true`) {
		t.Fatalf("repository projection: %d %s", repositoryHTTP.Code, repositoryHTTP.Body.String())
	}
}
