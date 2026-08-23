package toudocu

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const (
	reviewAPIBase           = "/_toudocu/api/agent"
	reviewRepositoryAPIBase = changesAPIBase + "/review"
)

var reviewRepositoryRouteRegistry = []apiRoute{
	{Path: reviewRepositoryAPIBase + "/repository/changes", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewRepositoryAPIBase + "/repository/files", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewRepositoryAPIBase + "/repository/file", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveReviewAPI},
}

var reviewRouteRegistry = []apiRoute{
	{Path: reviewAPIBase + "/discussions", Methods: []string{http.MethodGet, http.MethodPost}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewAPIBase + "/discussions/{discussionId}", Methods: []string{http.MethodGet, http.MethodPatch, http.MethodDelete}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewAPIBase + "/discussions/{discussionId}/messages", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewAPIBase + "/discussions/{discussionId}/messages/{messageId}", Methods: []string{http.MethodPatch, http.MethodDelete}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewAPIBase + "/deliveries/next", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveReviewAPI},
	{Path: reviewAPIBase + "/deliveries/{deliveryId}/response", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveReviewAPI},
}

func reviewOptionsFromRequest(base Options, request *http.Request) Options {
	options := base
	options.ChangeBase = request.URL.Query().Get("base")
	options.ChangeTarget = request.URL.Query().Get("target")
	options.ChangeBranchBase = request.URL.Query().Get("branchBase")
	return options
}

func (s *documentationServer) serveReviewAPI(w http.ResponseWriter, request *http.Request) {
	path := strings.TrimSuffix(request.URL.Path, "/")
	if strings.HasPrefix(path, reviewRepositoryAPIBase+"/") {
		s.serveReviewRepositoryAPI(w, request, path)
		return
	}
	options := reviewOptionsFromRequest(s.options, request)
	if options.ChangeTarget != "" && options.ChangeTarget != "working-tree" {
		writeReviewError(w, agentFailure("AGENT_INVALID_TARGET", http.StatusForbidden, "discussions are read-only outside target=working-tree"))
		return
	}
	service, err := newReviewService(options)
	if err != nil {
		writeReviewError(w, err)
		return
	}
	if path == reviewAPIBase+"/discussions" {
		switch request.Method {
		case http.MethodGet:
			state, loadErr := service.discussions()
			if loadErr != nil {
				writeReviewError(w, loadErr)
				return
			}
			etag := `"` + digestBytes([]byte(state.StateDigest+"\x00"+state.RepositoryRevision)) + `"`
			w.Header().Set("ETag", etag)
			if request.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			writeChangesJSON(w, http.StatusOK, state)
		case http.MethodPost:
			if !requireReviewJSONAction(w, request, "agent-discussion-create") {
				return
			}
			var input CreateDiscussionRequest
			if !decodeReviewJSON(w, request, &input) {
				return
			}
			state, createErr := service.createDiscussion(input)
			if createErr != nil {
				writeReviewError(w, createErr)
				return
			}
			writeChangesJSON(w, http.StatusCreated, state)
		default:
			allowReviewMethods(w, request, http.MethodGet, http.MethodPost)
		}
		return
	}
	if path == reviewAPIBase+"/deliveries/next" {
		if !allowReviewMethods(w, request, http.MethodPost) || !requireReviewJSONAction(w, request, "agent-delivery-next") {
			return
		}
		if request.ContentLength > 0 {
			var empty struct{}
			if !decodeReviewJSON(w, request, &empty) {
				return
			}
		}
		next, nextErr := service.claimNext()
		if nextErr != nil {
			writeReviewError(w, nextErr)
			return
		}
		writeChangesJSON(w, http.StatusOK, next)
		return
	}
	if id, ok := agentPathID(path, reviewAPIBase+"/deliveries/", "/response", "DEL"); ok {
		if !allowReviewMethods(w, request, http.MethodPost) || !requireReviewJSONAction(w, request, "agent-delivery-response") {
			return
		}
		var response AgentResponse
		if !decodeReviewJSON(w, request, &response) {
			return
		}
		if response.DeliveryID != id {
			writeReviewError(w, agentFailure("AGENT_INVALID_MESSAGE", http.StatusBadRequest, "delivery path ID does not match payload"))
			return
		}
		state, responseErr := service.respond(response)
		if responseErr != nil {
			writeReviewError(w, responseErr)
			return
		}
		writeChangesJSON(w, http.StatusOK, state)
		return
	}
	if discussionID, tail, ok := agentDiscussionPath(path); ok {
		s.serveDiscussionResource(w, request, service, discussionID, tail)
		return
	}
	writeReviewError(w, agentFailure("AGENT_DISCUSSION_NOT_FOUND", http.StatusNotFound, "agent feedback route not found"))
}

func (s *documentationServer) serveReviewRepositoryAPI(w http.ResponseWriter, request *http.Request, path string) {
	if path == reviewRepositoryAPIBase+"/repository/changes" {
		if !allowReviewMethods(w, request, http.MethodGet) {
			return
		}
		report, err := BuildRepositoryReview(reviewOptionsFromRequest(s.options, request))
		if err != nil {
			writeReviewError(w, err)
			return
		}
		etag := `"` + report.RepositoryRevision + `"`
		w.Header().Set("ETag", etag)
		if request.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		writeChangesJSON(w, http.StatusOK, report)
		return
	}
	if path == reviewRepositoryAPIBase+"/repository/files" {
		if !allowReviewMethods(w, request, http.MethodGet) {
			return
		}
		limit, err := reviewQueryLimit(request.URL.Query().Get("limit"))
		if err != nil {
			writeReviewError(w, err)
			return
		}
		g, err := openGitRepositorySource(s.options.RepositoryRoot, s.options.ChangeRenameSimilarity)
		if err != nil {
			writeReviewError(w, err)
			return
		}
		files, err := reviewInventory(g, request.URL.Query().Get("q"), limit)
		if err != nil {
			writeReviewError(w, err)
			return
		}
		writeChangesJSON(w, http.StatusOK, map[string]any{"schemaVersion": reviewSchemaVersion, "files": files})
		return
	}
	if path == reviewRepositoryAPIBase+"/repository/file" {
		if !allowReviewMethods(w, request, http.MethodGet) {
			return
		}
		detail, err := BuildRepositoryReviewFile(reviewOptionsFromRequest(s.options, request), request.URL.Query().Get("path"))
		if err != nil {
			writeReviewError(w, err)
			return
		}
		if detail.Documentation != nil && detail.Documentation.RenderedDiffAvailable {
			if detail.Before != nil {
				rendered := s.renderChangesMarkdown(*detail.Before)
				detail.RenderedBefore = &rendered
			}
			if detail.Current != nil {
				rendered := s.renderChangesMarkdown(*detail.Current)
				detail.RenderedCurrent = &rendered
			}
		}
		writeChangesJSON(w, http.StatusOK, detail)
		return
	}
	writeReviewError(w, agentFailure("AGENT_INVALID_PATH", http.StatusNotFound, "review repository route not found"))
}

func (s *documentationServer) serveDiscussionResource(w http.ResponseWriter, request *http.Request, service *reviewService, discussionID, tail string) {
	if tail == "" {
		switch request.Method {
		case http.MethodGet:
			state, err := service.discussions()
			if err != nil {
				writeReviewError(w, err)
				return
			}
			discussion := findDiscussion(&state, discussionID)
			if discussion == nil {
				writeReviewError(w, agentFailure("AGENT_DISCUSSION_NOT_FOUND", http.StatusNotFound, "discussion not found"))
				return
			}
			writeChangesJSON(w, http.StatusOK, discussion)
		case http.MethodPatch:
			if !requireReviewJSONAction(w, request, "agent-discussion-update") {
				return
			}
			var input UpdateDiscussionRequest
			if !decodeReviewJSON(w, request, &input) {
				return
			}
			state, err := service.updateDiscussion(discussionID, input)
			if err != nil {
				writeReviewError(w, err)
				return
			}
			writeChangesJSON(w, http.StatusOK, state)
		case http.MethodDelete:
			if !requireReviewJSONAction(w, request, "agent-discussion-delete") {
				return
			}
			var guard ReviewMutationGuard
			if !decodeReviewJSON(w, request, &guard) {
				return
			}
			state, err := service.deleteDiscussion(discussionID, guard)
			if err != nil {
				writeReviewError(w, err)
				return
			}
			writeChangesJSON(w, http.StatusOK, state)
		default:
			allowReviewMethods(w, request, http.MethodGet, http.MethodPatch, http.MethodDelete)
		}
		return
	}
	if tail == "/messages" {
		if !allowReviewMethods(w, request, http.MethodPost) || !requireReviewJSONAction(w, request, "agent-message-create") {
			return
		}
		var input CreateMessageRequest
		if !decodeReviewJSON(w, request, &input) {
			return
		}
		state, err := service.createMessage(discussionID, input)
		if err != nil {
			writeReviewError(w, err)
			return
		}
		writeChangesJSON(w, http.StatusCreated, state)
		return
	}
	messageID, ok := agentMessageTail(tail)
	if !ok {
		writeReviewError(w, agentFailure("AGENT_INVALID_MESSAGE", http.StatusNotFound, "message route not found"))
		return
	}
	switch request.Method {
	case http.MethodPatch:
		if !requireReviewJSONAction(w, request, "agent-message-update") {
			return
		}
		var input UpdateMessageRequest
		if !decodeReviewJSON(w, request, &input) {
			return
		}
		state, err := service.updateMessage(discussionID, messageID, input)
		if err != nil {
			writeReviewError(w, err)
			return
		}
		writeChangesJSON(w, http.StatusOK, state)
	case http.MethodDelete:
		if !requireReviewJSONAction(w, request, "agent-message-delete") {
			return
		}
		var guard ReviewMutationGuard
		if !decodeReviewJSON(w, request, &guard) {
			return
		}
		state, err := service.deleteMessage(discussionID, messageID, guard)
		if err != nil {
			writeReviewError(w, err)
			return
		}
		writeChangesJSON(w, http.StatusOK, state)
	default:
		allowReviewMethods(w, request, http.MethodPatch, http.MethodDelete)
	}
}

func agentDiscussionPath(path string) (string, string, bool) {
	prefix := reviewAPIBase + "/discussions/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	remainder := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(remainder, "/", 2)
	if !validAgentID(parts[0], "DISC") {
		return "", "", false
	}
	tail := ""
	if len(parts) == 2 {
		tail = "/" + parts[1]
	}
	return parts[0], tail, true
}

func agentMessageTail(tail string) (string, bool) {
	prefix := "/messages/"
	if !strings.HasPrefix(tail, prefix) {
		return "", false
	}
	remainder := strings.TrimPrefix(tail, prefix)
	if strings.Contains(remainder, "/") || !validAgentID(remainder, "MSG") {
		return "", false
	}
	return remainder, true
}

func agentPathID(path, prefix, suffix, kind string) (string, bool) {
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	return id, validAgentID(id, kind)
}

func validAgentID(id, kind string) bool {
	return len(id) == len(kind)+1+26 && strings.HasPrefix(id, kind+"-") && !strings.Contains(id, "/")
}

func allowReviewMethods(w http.ResponseWriter, request *http.Request, methods ...string) bool {
	for _, method := range methods {
		if request.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeReviewError(w, agentFailure("AGENT_INVALID_MESSAGE", http.StatusMethodNotAllowed, "method not allowed"))
	return false
}

func requireReviewJSONAction(w http.ResponseWriter, request *http.Request, action string) bool {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeReviewError(w, agentFailure("AGENT_INVALID_MESSAGE", http.StatusUnsupportedMediaType, "Content-Type application/json is required"))
		return false
	}
	if request.Header.Get("X-Toudocu-Action") != action || !editorOriginAllowed(request) {
		writeReviewError(w, agentFailure("AGENT_INVALID_MESSAGE", http.StatusForbidden, "mutation requires the expected action and same origin"))
		return false
	}
	return true
}

func decodeReviewJSON(w http.ResponseWriter, request *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, request.Body, reviewRequestLimit)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		status, code := http.StatusBadRequest, "AGENT_INVALID_MESSAGE"
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			status, code = http.StatusRequestEntityTooLarge, "AGENT_PAYLOAD_TOO_LARGE"
		}
		writeReviewError(w, agentFailure(code, status, "invalid agent feedback JSON: "+err.Error()))
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeReviewError(w, agentFailure("AGENT_INVALID_MESSAGE", http.StatusBadRequest, "request body must contain one JSON object"))
		return false
	}
	return true
}

func writeReviewError(w http.ResponseWriter, err error) {
	failure := &reviewFailure{Code: "AGENT_STATE_CORRUPTED", Status: http.StatusInternalServerError, Message: err.Error()}
	var typed *reviewFailure
	if errors.As(err, &typed) {
		failure = typed
	}
	writeChangesJSON(w, failure.Status, map[string]any{"schemaVersion": reviewSchemaVersion, "diagnostics": []Issue{{Severity: "error", Code: failure.Code, Message: failure.Message}}})
}

func reviewQueryLimit(raw string) (int, error) {
	if raw == "" {
		return 50, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 200 {
		return 0, agentFailure("AGENT_INVALID_MESSAGE", http.StatusBadRequest, "limit must be between 1 and 200")
	}
	return value, nil
}
