package toudocu

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type changeFailure struct {
	Code      int
	IssueCode string
	Err       error
}

func (e *changeFailure) Error() string { return e.Err.Error() }
func (e *changeFailure) Unwrap() error { return e.Err }

type gitChangeSource struct {
	root        string
	docsRoot    string
	docsRel     string
	similarity  int
	status      []byte
	statusErr   error
	statusRead  bool
	cacheStatus bool
}

type gitFileChange struct {
	status, path, oldPath string
	state                 ChangeGitState
}

func openGitChangeSource(docsRoot string, similarity int) (*gitChangeSource, error) {
	absDocs, err := filepath.Abs(docsRoot)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "-C", absDocs, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, &changeFailure{Code: 3, IssueCode: "git-command-failed", Err: fmt.Errorf("git is unavailable")}
		}
		return nil, &changeFailure{Code: 3, IssueCode: "git-repository-not-found", Err: fmt.Errorf("documentation directory is not inside a Git repository")}
	}
	root := strings.TrimSpace(string(out))
	canonicalRoot, err := resolvePathForSafety(root)
	if err != nil {
		return nil, err
	}
	canonicalDocs, err := resolvePathForSafety(absDocs)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(canonicalRoot, canonicalDocs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, &changeFailure{Code: 2, IssueCode: "git-path-outside-documentation-root", Err: fmt.Errorf("documentation directory is outside the Git root")}
	}
	if similarity < 1 || similarity > 100 {
		similarity = 60
	}
	return &gitChangeSource{root: canonicalRoot, docsRoot: canonicalDocs, docsRel: filepath.ToSlash(rel), similarity: similarity}, nil
}

func (g *gitChangeSource) run(args ...string) ([]byte, error) {
	commandArgs := []string{"-C", g.root, "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false"}
	cmd := exec.Command("git", append(commandArgs, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "LANG=C")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", args[0], message)
	}
	return out, nil
}

func (g *gitChangeSource) statusSnapshot() ([]byte, error) {
	if !g.cacheStatus {
		return g.run("status", "--porcelain=v2", "-z", "--untracked-files=all", "--", g.docsRel)
	}
	if !g.statusRead {
		g.status, g.statusErr = g.run("status", "--porcelain=v2", "-z", "--untracked-files=all", "--", g.docsRel)
		g.statusRead = true
	}
	return g.status, g.statusErr
}

func validChangeRevision(value string) bool {
	if value == "" || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\x00\r\n") {
		return false
	}
	return !strings.Contains(value, "..") && !strings.Contains(value, ":") && !strings.ContainsAny(value, "?*[\\")
}

func (g *gitChangeSource) resolveCommit(ref string) (string, error) {
	if !validChangeRevision(ref) {
		return "", &changeFailure{Code: 2, IssueCode: "git-base-not-found", Err: fmt.Errorf("invalid Git revision: %q", ref)}
	}
	out, err := g.run("rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", &changeFailure{Code: 2, IssueCode: "git-base-not-found", Err: fmt.Errorf("git revision %q not found", ref)}
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *gitChangeSource) repositoryState() (ChangeRepository, error) {
	head, err := g.resolveCommit("HEAD")
	if err != nil {
		return ChangeRepository{}, err
	}
	branchBytes, _ := g.run("symbolic-ref", "--quiet", "--short", "HEAD")
	status, err := g.statusSnapshot()
	if err != nil {
		return ChangeRepository{}, err
	}
	return ChangeRepository{Root: g.root, Branch: strings.TrimSpace(string(branchBytes)), Head: shortObjectID(head), Dirty: len(status) > 0}, nil
}

func shortObjectID(value string) string {
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func (g *gitChangeSource) statusStates() (map[string]ChangeGitState, error) {
	out, err := g.statusSnapshot()
	if err != nil {
		return nil, err
	}
	states := map[string]ChangeGitState{}
	tokens := bytes.Split(out, []byte{0})
	for i := 0; i < len(tokens); i++ {
		record := string(tokens[i])
		if record == "" {
			continue
		}
		recordType := record[:1]
		switch recordType {
		case "?":
			path := strings.TrimPrefix(record, "? ")
			states[filepath.ToSlash(path)] = ChangeGitState{Untracked: true, Unstaged: true}
		case "1":
			fields := strings.SplitN(record, " ", 9)
			if len(fields) != 9 {
				continue
			}
			xy := fields[1]
			path := fields[8]
			state := ChangeGitState{Staged: len(xy) > 0 && xy[0] != '.', Unstaged: len(xy) > 1 && xy[1] != '.'}
			states[filepath.ToSlash(path)] = state
		case "2":
			fields := strings.SplitN(record, " ", 10)
			if len(fields) == 10 {
				xy := fields[1]
				path := fields[9]
				state := ChangeGitState{Staged: len(xy) > 0 && xy[0] != '.', Unstaged: len(xy) > 1 && xy[1] != '.'}
				states[filepath.ToSlash(path)] = state
			}
			i++ // The following NUL token is the original path.
		}
	}
	return states, nil
}

func parseNameStatus(data []byte) ([]gitFileChange, error) {
	tokens := bytes.Split(data, []byte{0})
	out := []gitFileChange{}
	for i := 0; i < len(tokens); {
		if len(tokens[i]) == 0 {
			i++
			continue
		}
		statusToken := string(tokens[i])
		i++
		var inlinePath string
		if parts := strings.SplitN(statusToken, "\t", 2); len(parts) == 2 {
			statusToken, inlinePath = parts[0], parts[1]
		}
		if statusToken == "" {
			continue
		}
		code := statusToken[:1]
		readPath := func() (string, error) {
			if inlinePath != "" {
				p := inlinePath
				inlinePath = ""
				return filepath.ToSlash(p), nil
			}
			if i >= len(tokens) || len(tokens[i]) == 0 {
				return "", fmt.Errorf("incomplete NUL-separated name-status")
			}
			p := filepath.ToSlash(string(tokens[i]))
			i++
			return p, nil
		}
		first, err := readPath()
		if err != nil {
			return nil, err
		}
		change := gitFileChange{status: mapGitStatus(code), path: first}
		if code == "R" || code == "C" {
			second, err := readPath()
			if err != nil {
				return nil, err
			}
			change.oldPath, change.path = first, second
		}
		out = append(out, change)
	}
	return out, nil
}

func mapGitStatus(code string) string {
	switch code {
	case "A":
		return "added"
	case "M":
		return "modified"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	case "T":
		return "type-changed"
	}
	return "modified"
}

func (g *gitChangeSource) listChanges(base ChangeSide, target ChangeSide) ([]gitFileChange, error) {
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--name-status", "-z", fmt.Sprintf("--find-renames=%d%%", g.similarity), fmt.Sprintf("--find-copies=%d%%", g.similarity)}
	switch target.Type {
	case "working-tree":
		if base.Type != "index" {
			args = append(args, base.Resolved)
		}
	case "index":
		args = append(args, "--cached", base.Resolved)
	case "commit":
		args = append(args, base.Resolved, target.Resolved)
	default:
		return nil, fmt.Errorf("unsupported target %q", target.Type)
	}
	args = append(args, "--", g.docsRel)
	out, err := g.run(args...)
	if err != nil {
		return nil, err
	}
	changes, err := parseNameStatus(out)
	if err != nil {
		return nil, err
	}
	states, err := g.statusStates()
	if err != nil {
		return nil, err
	}
	for i := range changes {
		changes[i].state = states[changes[i].path]
	}
	if base.Type == "commit" && target.Type == "working-tree" {
		committed, committedErr := g.committedPaths(base.Resolved)
		if committedErr != nil {
			return nil, committedErr
		}
		for i := range changes {
			changes[i].state.CommittedInBranch = committed[changes[i].path] || committed[changes[i].oldPath]
		}
	}
	if target.Type == "working-tree" {
		untracked, err := g.run("ls-files", "--others", "--exclude-standard", "-z", "--", g.docsRel)
		if err != nil {
			return nil, err
		}
		known := map[string]bool{}
		for _, c := range changes {
			known[c.path] = true
		}
		for _, token := range bytes.Split(untracked, []byte{0}) {
			path := filepath.ToSlash(string(token))
			if path == "" || known[path] {
				continue
			}
			changes = append(changes, gitFileChange{status: "untracked", path: path, state: ChangeGitState{Untracked: true, Unstaged: true}})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].path < changes[j].path })
	return changes, nil
}

func (g *gitChangeSource) committedPaths(base string) (map[string]bool, error) {
	if base == "" {
		return map[string]bool{}, nil
	}
	out, err := g.run("diff", "--no-ext-diff", "--no-textconv", "--no-color", "--name-only", "-z", base, "HEAD", "--", g.docsRel)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for _, token := range bytes.Split(out, []byte{0}) {
		if path := filepath.ToSlash(string(token)); path != "" {
			paths[path] = true
		}
	}
	return paths, nil
}

func (g *gitChangeSource) diff(base, target ChangeSide, change gitFileChange) ([]byte, error) {
	if change.status == "untracked" {
		content, err := os.ReadFile(filepath.Join(g.root, filepath.FromSlash(change.path)))
		if err != nil {
			return nil, err
		}
		return addedFilePatch(change.path, content), nil
	}
	args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--full-index", "--binary", "--unified=3"}
	switch target.Type {
	case "working-tree":
		if base.Type != "index" {
			args = append(args, base.Resolved)
		}
	case "index":
		args = append(args, "--cached", base.Resolved)
	case "commit":
		args = append(args, base.Resolved, target.Resolved)
	}
	args = append(args, "--")
	if change.oldPath != "" {
		args = append(args, change.oldPath)
	}
	args = append(args, change.path)
	return g.run(args...)
}

func addedFilePatch(path string, content []byte) []byte {
	var out strings.Builder
	_, _ = fmt.Fprintf(&out, "diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n", path, path, path)
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	count := len(lines)
	if count > 0 && lines[count-1] == "" {
		count--
	}
	_, _ = fmt.Fprintf(&out, "@@ -0,0 +1,%d @@\n", count)
	for i := 0; i < count; i++ {
		out.WriteByte('+')
		out.WriteString(lines[i])
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func (g *gitChangeSource) content(side ChangeSide, path string) ([]byte, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	if filepath.IsAbs(path) || strings.Contains(path, "\\") || path == ".." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
		return nil, fmt.Errorf("invalid path")
	}
	switch side.Type {
	case "working-tree":
		return os.ReadFile(filepath.Join(g.root, filepath.FromSlash(path)))
	case "index":
		return g.run("show", ":"+path)
	case "commit":
		return g.run("cat-file", "blob", side.Resolved+":"+path)
	default:
		return nil, fmt.Errorf("unsupported side %q", side.Type)
	}
}

func taskIDFromContent(content []byte) string {
	return strings.TrimSpace(analyzeMarkdown(string(content)).Metadata["id"])
}

func (g *gitChangeSource) taskDocuments(side ChangeSide, taskID string) (map[string]changeTaskDocument, error) {
	workPath := filepath.ToSlash(filepath.Join(g.docsRel, "work"))
	var paths []string
	switch side.Type {
	case "working-tree":
		root := filepath.Join(g.root, filepath.FromSlash(workPath))
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				return nil
			}
			rel, relErr := filepath.Rel(g.root, path)
			if relErr != nil {
				return relErr
			}
			paths = append(paths, filepath.ToSlash(rel))
			return nil
		}); err != nil {
			return nil, err
		}
	case "index":
		out, err := g.run("ls-files", "-z", "--", workPath)
		if err != nil {
			return nil, err
		}
		for _, raw := range bytes.Split(out, []byte{0}) {
			path := filepath.ToSlash(string(raw))
			if strings.HasSuffix(strings.ToLower(path), ".md") {
				paths = append(paths, path)
			}
		}
	case "commit":
		out, err := g.run("ls-tree", "-r", "-z", "--name-only", side.Resolved, "--", workPath)
		if err != nil {
			return nil, err
		}
		for _, raw := range bytes.Split(out, []byte{0}) {
			path := filepath.ToSlash(string(raw))
			if strings.HasSuffix(strings.ToLower(path), ".md") {
				paths = append(paths, path)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported side %q", side.Type)
	}
	sort.Strings(paths)
	tasks := map[string]changeTaskDocument{}
	for _, path := range paths {
		content, err := g.content(side, path)
		if err == nil {
			id := taskIDFromContent(content)
			if id == "" || taskID != "" && id != taskID {
				continue
			}
			if previous, exists := tasks[id]; exists {
				return nil, fmt.Errorf("task identifier %s is ambiguous (%s and %s)", id, previous.path, path)
			}
			tasks[id] = changeTaskDocument{path: path, content: content}
			continue
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	return tasks, nil
}

func isBinaryContent(content []byte) bool {
	return bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content)
}
