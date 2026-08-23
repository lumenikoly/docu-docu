package toudocu

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func openGitRepositorySource(repositoryRoot string, similarity int) (*gitChangeSource, error) {
	if strings.TrimSpace(repositoryRoot) == "" {
		repositoryRoot = "."
	}
	abs, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return nil, err
	}
	canonical, err := resolvePathForSafety(abs)
	if err != nil {
		return nil, err
	}
	probe := execGit(canonical, "rev-parse", "--show-toplevel")
	if probe.err != nil {
		return nil, &reviewFailure{Code: "REVIEW_GIT_UNAVAILABLE", Status: http.StatusServiceUnavailable, Message: "repository root is not inside an available Git repository"}
	}
	gitRoot, err := resolvePathForSafety(strings.TrimSpace(string(probe.out)))
	if err != nil || filepath.Clean(gitRoot) != filepath.Clean(canonical) {
		return nil, &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "--repository-root must point to the canonical Git root"}
	}
	if similarity < 1 || similarity > 100 {
		similarity = 60
	}
	return &gitChangeSource{root: canonical, docsRoot: canonical, docsRel: ".", similarity: similarity}, nil
}

type gitExecResult struct {
	out []byte
	err error
}

func execGit(root string, args ...string) gitExecResult {
	g := &gitChangeSource{root: root}
	out, err := g.run(args...)
	return gitExecResult{out: out, err: err}
}

func resolveReviewComparison(g *gitChangeSource, options Options) (ChangeSide, ChangeSide, error) {
	baseRef := strings.TrimSpace(options.ChangeBase)
	if baseRef == "" {
		baseRef = "HEAD"
	}
	base := ChangeSide{Type: "commit", Revision: baseRef, DisplayRef: baseRef}
	var err error
	if baseRef == "index" {
		base.Type = "index"
	} else {
		base.Resolved, err = g.resolveCommit(baseRef)
	}
	if err != nil {
		return ChangeSide{}, ChangeSide{}, err
	}
	if strings.TrimSpace(options.ChangeBranchBase) != "" {
		branch := strings.TrimSpace(options.ChangeBranchBase)
		branchCommit, branchErr := g.resolveCommit(branch)
		if branchErr != nil {
			return ChangeSide{}, ChangeSide{}, branchErr
		}
		head, headErr := g.resolveCommit("HEAD")
		if headErr != nil {
			return ChangeSide{}, ChangeSide{}, headErr
		}
		mergeBase, mergeErr := g.run("merge-base", branchCommit, head)
		if mergeErr != nil {
			return ChangeSide{}, ChangeSide{}, &reviewFailure{Code: "REVIEW_GIT_UNAVAILABLE", Status: http.StatusServiceUnavailable, Message: "merge-base is unavailable"}
		}
		base.Revision, base.Resolved, base.DisplayRef = branch, strings.TrimSpace(string(mergeBase)), "merge-base("+branch+", HEAD)"
	}
	targetValue := strings.TrimSpace(options.ChangeTarget)
	if targetValue == "" {
		targetValue = "working-tree"
	}
	target := ChangeSide{Type: targetValue, DisplayRef: targetValue}
	switch targetValue {
	case "working-tree", "index":
	case "HEAD":
		target.Type, target.Revision, target.DisplayRef = "commit", "HEAD", "HEAD"
		target.Resolved, err = g.resolveCommit("HEAD")
	default:
		target.Type, target.Revision = "commit", targetValue
		target.Resolved, err = g.resolveCommit(targetValue)
	}
	if err != nil {
		return ChangeSide{}, ChangeSide{}, err
	}
	return base, target, nil
}

func repositoryReviewRevision(g *gitChangeSource, base, target ChangeSide) (string, error) {
	head, err := g.resolveCommit("HEAD")
	if err != nil {
		return "", err
	}
	status, err := g.statusSnapshot()
	if err != nil {
		return "", err
	}
	diff, err := g.run("diff", "--no-ext-diff", "--no-textconv", "--no-color", "--full-index", "--binary", "HEAD", "--", ".")
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, part := range []string{base.Type, base.Revision, base.Resolved, target.Type, target.Revision, target.Resolved, head} {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	_, _ = hash.Write(status)
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(diff)
	untracked, err := g.run("ls-files", "--others", "--exclude-standard", "-z", "--", ".")
	if err != nil {
		return "", err
	}
	for _, token := range bytes.Split(untracked, []byte{0}) {
		path := filepath.ToSlash(string(token))
		if path == "" {
			continue
		}
		validated, pathErr := validateReviewPath(g, path)
		if pathErr != nil {
			return "", pathErr
		}
		if pathErr := ensureReviewPathSafe(g, ChangeSide{Type: "working-tree"}, validated); pathErr != nil {
			return "", pathErr
		}
		file, openErr := os.Open(filepath.Join(g.root, filepath.FromSlash(validated)))
		if openErr != nil {
			return "", openErr
		}
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(validated))
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func BuildRepositoryReview(options Options) (*RepositoryReviewReport, error) {
	g, err := openGitRepositorySource(options.RepositoryRoot, options.ChangeRenameSimilarity)
	if err != nil {
		return nil, err
	}
	g.cacheStatus = true
	base, target, err := resolveReviewComparison(g, options)
	if err != nil {
		return nil, err
	}
	repository, err := g.repositoryState()
	if err != nil {
		return nil, &reviewFailure{Code: "REVIEW_GIT_UNAVAILABLE", Status: http.StatusServiceUnavailable, Message: err.Error()}
	}
	revision, err := repositoryReviewRevision(g, base, target)
	if err != nil {
		return nil, err
	}
	changed, err := g.listChanges(base, target)
	if err != nil {
		return nil, &reviewFailure{Code: "REVIEW_GIT_UNAVAILABLE", Status: http.StatusServiceUnavailable, Message: err.Error()}
	}
	report := &RepositoryReviewReport{
		SchemaVersion: reviewSchemaVersion, Repository: repository,
		Comparison: ChangeComparison{Base: base, Target: target}, RepositoryRevision: revision,
		Summary: ChangeSummary{Entities: map[string]int{}, Classifications: map[string]int{}},
		Files:   []RepositoryReviewFile{}, FeedbackWritable: target.Type == "working-tree", Diagnostics: []Issue{},
	}
	documentation := documentationChangesByPath(options)
	requested := filepath.ToSlash(strings.TrimSpace(options.ChangeFile))
	for _, change := range changed {
		if requested != "" && requested != change.path && requested != change.oldPath {
			continue
		}
		file := RepositoryReviewFile{Status: change.status, Path: change.path, OldPath: change.oldPath, GitState: change.state, Language: reviewLanguage(change.path)}
		patch, patchErr := g.diff(base, target, change)
		if patchErr == nil {
			file.Lines = countPatchLines(string(patch))
		}
		contentPath := change.path
		if target.Type != "working-tree" && change.status == "deleted" {
			contentPath = change.oldPath
			if contentPath == "" {
				contentPath = change.path
			}
		}
		if change.status != "deleted" {
			if content, contentErr := readReviewContent(g, target, contentPath); contentErr == nil {
				file.Size = int64(len(content))
				file.Binary = isBinaryContent(content)
				if !file.Binary {
					file.Digest = digestBytes(content)
				}
			}
		}
		if doc := documentation[file.Path]; doc != nil {
			copy := *doc
			file.Documentation = &copy
		} else if file.OldPath != "" {
			if doc := documentation[file.OldPath]; doc != nil {
				copy := *doc
				file.Documentation = &copy
			}
		}
		report.Files = append(report.Files, file)
		addReviewSummary(&report.Summary, file)
	}
	return report, nil
}

func documentationChangesByPath(options Options) map[string]*DocumentationChange {
	result := map[string]*DocumentationChange{}
	if strings.TrimSpace(options.InputDirectory) == "" {
		return result
	}
	report, err := BuildDocumentationChanges(options)
	if err != nil {
		return result
	}
	for index := range report.Changes {
		change := &report.Changes[index]
		result[change.Path] = change
		if change.OldPath != "" {
			result[change.OldPath] = change
		}
	}
	return result
}

func addReviewSummary(summary *ChangeSummary, file RepositoryReviewFile) {
	switch file.Status {
	case "added":
		summary.Files.Added++
	case "modified":
		summary.Files.Modified++
	case "deleted":
		summary.Files.Deleted++
	case "renamed":
		summary.Files.Renamed++
	case "copied":
		summary.Files.Copied++
	case "type-changed":
		summary.Files.TypeChanged++
	case "untracked":
		summary.Files.Untracked++
	}
	summary.Lines.Added += file.Lines.Added
	summary.Lines.Deleted += file.Lines.Deleted
}

func reviewInventory(g *gitChangeSource, query string, limit int) ([]RepositoryReviewFile, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	out, err := g.run("ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", ".")
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	seen := map[string]bool{}
	files := []RepositoryReviewFile{}
	for _, token := range bytes.Split(out, []byte{0}) {
		path := filepath.ToSlash(string(token))
		if path == "" || seen[path] || path == ".git" || strings.HasPrefix(path, ".git/") {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(path), query) && !strings.Contains(strings.ToLower(filepath.Base(path)), query) {
			continue
		}
		if safeErr := ensureReviewPathSafe(g, ChangeSide{Type: "working-tree"}, path); safeErr != nil {
			continue
		}
		info, statErr := os.Stat(filepath.Join(g.root, filepath.FromSlash(path)))
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		seen[path] = true
		files = append(files, RepositoryReviewFile{Path: path, Size: info.Size(), Language: reviewLanguage(path)})
		if len(files) >= limit {
			break
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func BuildRepositoryReviewFile(options Options, requested string) (*RepositoryReviewFileDetail, error) {
	options.ChangeFile = requested
	report, err := BuildRepositoryReview(options)
	if err != nil {
		return nil, err
	}
	g, err := openGitRepositorySource(options.RepositoryRoot, options.ChangeRenameSimilarity)
	if err != nil {
		return nil, err
	}
	path, err := validateReviewPath(g, requested)
	if err != nil {
		return nil, err
	}
	var selected *RepositoryReviewFile
	for index := range report.Files {
		if report.Files[index].Path == path || report.Files[index].OldPath == path {
			selected = &report.Files[index]
			break
		}
	}
	if selected == nil {
		selected = &RepositoryReviewFile{Path: path, Status: "linked", Language: reviewLanguage(path)}
	}
	detail := &RepositoryReviewFileDetail{SchemaVersion: reviewSchemaVersion, RepositoryRevision: report.RepositoryRevision, File: *selected, Hunks: []SourceDiffHunk{}, Documentation: selected.Documentation}
	if selected.Status == "linked" {
		content, readErr := readReviewText(g, ChangeSide{Type: "working-tree"}, path)
		if readErr != nil {
			return nil, readErr
		}
		value := string(content)
		detail.Current = &value
		detail.File.Size, detail.File.Digest = int64(len(content)), digestBytes(content)
		return detail, nil
	}
	change := gitFileChange{status: selected.Status, path: selected.Path, oldPath: selected.OldPath, state: selected.GitState}
	if selected.Status != "added" && selected.Status != "untracked" {
		beforePath := selected.Path
		if selected.OldPath != "" {
			beforePath = selected.OldPath
		}
		content, readErr := readReviewText(g, report.Comparison.Base, beforePath)
		if readErr != nil {
			return nil, readErr
		}
		value := string(content)
		detail.Before = &value
	}
	if selected.Status != "deleted" {
		content, readErr := readReviewText(g, report.Comparison.Target, selected.Path)
		if readErr != nil {
			return nil, readErr
		}
		value := string(content)
		detail.Current = &value
	}
	patch, patchErr := g.diff(report.Comparison.Base, report.Comparison.Target, change)
	if patchErr != nil {
		return nil, patchErr
	}
	detail.Patch = string(patch)
	detail.Hunks = parseSourceDiffHunks(detail.Patch)
	return detail, nil
}

func readReviewText(g *gitChangeSource, side ChangeSide, path string) ([]byte, error) {
	content, err := readReviewContent(g, side, path)
	if err != nil {
		return nil, err
	}
	if len(content) > reviewSnapshotLimit {
		return nil, &reviewFailure{Code: "REVIEW_TOO_LARGE", Status: http.StatusRequestEntityTooLarge, Message: "review source exceeds 2 MiB"}
	}
	if isBinaryContent(content) {
		return nil, &reviewFailure{Code: "REVIEW_BINARY", Status: http.StatusUnsupportedMediaType, Message: "binary review source is not supported"}
	}
	return content, nil
}

func readReviewContent(g *gitChangeSource, side ChangeSide, path string) ([]byte, error) {
	if err := ensureReviewPathSafe(g, side, path); err != nil {
		return nil, err
	}
	content, err := g.content(side, path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &reviewFailure{Code: "REVIEW_NOT_FOUND", Status: http.StatusNotFound, Message: "review file not found"}
		}
		return nil, err
	}
	return content, nil
}

func validateReviewPath(g *gitChangeSource, requested string) (string, error) {
	path := filepath.ToSlash(strings.TrimSpace(requested))
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if path == "" || path != clean || filepath.IsAbs(path) || strings.Contains(path, "\\") || strings.Contains(path, "%") || path == ".." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") || path == ".git" || strings.HasPrefix(path, ".git/") {
		return "", &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "unsafe repository-relative path"}
	}
	if !pathContains(g.root, filepath.Join(g.root, filepath.FromSlash(path))) {
		return "", &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "path escapes the repository root"}
	}
	return path, nil
}

func ensureReviewPathSafe(g *gitChangeSource, side ChangeSide, requested string) error {
	path, err := validateReviewPath(g, requested)
	if err != nil {
		return err
	}
	if side.Type == "working-tree" {
		current := g.root
		parts := strings.Split(path, "/")
		for _, part := range parts {
			current = filepath.Join(current, part)
			info, statErr := os.Lstat(current)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					return &reviewFailure{Code: "REVIEW_NOT_FOUND", Status: http.StatusNotFound, Message: "review file not found"}
				}
				return statErr
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "symbolic-link or reparse path is forbidden"}
			}
		}
		info, statErr := os.Lstat(current)
		if statErr != nil || !info.Mode().IsRegular() {
			return &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "review path must be a regular file"}
		}
		return nil
	}
	var mode []byte
	switch side.Type {
	case "index":
		mode, err = g.run("ls-files", "-s", "--", path)
	case "commit":
		mode, err = g.run("ls-tree", side.Resolved, "--", path)
	default:
		return &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "unknown review side"}
	}
	if err != nil || len(mode) == 0 {
		return &reviewFailure{Code: "REVIEW_NOT_FOUND", Status: http.StatusNotFound, Message: "review file not found"}
	}
	fields := strings.Fields(string(mode))
	if len(fields) == 0 || fields[0] != "100644" && fields[0] != "100755" {
		return &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "symbolic-link or non-regular Git entry is forbidden"}
	}
	return nil
}

func validateReviewChangedPath(g *gitChangeSource, requested string) error {
	path, err := validateReviewPath(g, requested)
	if err != nil {
		return err
	}
	current := g.root
	parts := strings.Split(path, "/")
	for index, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			return nil // Deleted paths remain valid hints.
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "symbolic-link or reparse changedPath is forbidden"}
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return &reviewFailure{Code: "REVIEW_UNSAFE_PATH", Status: http.StatusForbidden, Message: "changedPath must be a regular file or a deleted path"}
		}
	}
	return nil
}

func reviewLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".java":
		return "java"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx", ".mts", ".cts":
		return "typescript"
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "text"
	}
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
