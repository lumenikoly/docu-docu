package toudocu

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const editorContentLimit = 2 << 20

type editorDiagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

type editorFile struct {
	Path        string             `json:"path"`
	Language    string             `json:"language"`
	Size        int64              `json:"size"`
	Digest      string             `json:"digest"`
	Title       string             `json:"title,omitempty"`
	DocumentURL string             `json:"documentURL,omitempty"`
	Content     string             `json:"content"`
	Diagnostics []editorDiagnostic `json:"diagnostics"`
}

type editorFileSummary struct {
	Path        string `json:"path"`
	Language    string `json:"language"`
	Size        int64  `json:"size"`
	Digest      string `json:"digest"`
	Title       string `json:"title,omitempty"`
	DocumentURL string `json:"documentURL,omitempty"`
}

func (f editorFile) summary() editorFileSummary {
	return editorFileSummary{Path: f.Path, Language: f.Language, Size: f.Size, Digest: f.Digest, Title: f.Title, DocumentURL: f.DocumentURL}
}

type editorWorkspace struct {
	root           string
	repositoryRoot string
	output         string
	excludes       map[string]struct{}
	modelOptions   Options
	outputRelative string
	cache          map[string]editorFileCache
}

type editorFileCache struct {
	size     int64
	modified int64
	digest   string
	title    string
}

type workspaceError struct {
	code string
	err  error
}

func (e *workspaceError) Error() string { return e.err.Error() }
func (e *workspaceError) Unwrap() error { return e.err }

func workspaceFailure(code, format string, args ...any) error {
	return &workspaceError{code: code, err: fmt.Errorf(format, args...)}
}

func workspaceErrorCode(err error) string {
	var target *workspaceError
	if errors.As(err, &target) {
		return target.code
	}
	return "workspace_error"
}

func newEditorWorkspace(options Options) (*editorWorkspace, error) {
	root, err := filepath.Abs(options.InputDirectory)
	if err != nil {
		return nil, err
	}
	output, err := filepath.Abs(options.OutputDirectory)
	if err != nil {
		return nil, err
	}
	repositoryRoot := options.RepositoryRoot
	if repositoryRoot == "" {
		repositoryRoot = filepath.Dir(root)
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return nil, err
	}
	excludes := map[string]struct{}{}
	for _, value := range append(append([]string{}, defaultExcludes...), options.Excludes...) {
		if value = strings.TrimSpace(value); value != "" {
			excludes[normalizeSlashes(value)] = struct{}{}
		}
	}
	outputRelative := ""
	if ensureInside(root, output) {
		outputRelative = toPosixRelative(root, output)
	}
	return &editorWorkspace{root: root, repositoryRoot: repositoryRoot, output: output, excludes: excludes, modelOptions: options, outputRelative: outputRelative, cache: map[string]editorFileCache{}}, nil
}

func editorLanguage(filePath string) (string, bool) {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".md":
		return "markdown", true
	case ".json":
		return "json", true
	case ".yaml", ".yml":
		return "yaml", true
	default:
		return "", false
	}
}

func contentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (w *editorWorkspace) excluded(relative, base string) bool {
	if strings.HasPrefix(base, ".") || shouldExclude(relative, base, w.excludes) {
		return true
	}
	return w.outputRelative != "" && (relative == w.outputRelative || strings.HasPrefix(relative, w.outputRelative+"/"))
}

func (w *editorWorkspace) scan(model *Model) ([]editorFile, string, error) {
	files := []editorFile{}
	nextCache := map[string]editorFileCache{}
	err := filepath.WalkDir(w.root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == w.root {
			return nil
		}
		relative := toPosixRelative(w.root, filePath)
		if filepath.Clean(w.root) == filepath.Clean(w.repositoryRoot) && relative == projectChangelogFile {
			return nil
		}
		info, err := os.Lstat(filePath)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if w.excluded(relative, entry.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		language, supported := editorLanguage(relative)
		if info.IsDir() || !info.Mode().IsRegular() || !supported {
			return nil
		}
		cached, unchanged := w.cache[relative]
		unchanged = unchanged && cached.size == info.Size() && cached.modified == info.ModTime().UnixNano()
		if !unchanged {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			cached = editorFileCache{size: int64(len(content)), modified: info.ModTime().UnixNano(), digest: contentDigest(content)}
			if language == "markdown" {
				cached.title = analyzeMarkdown(string(content)).Title
			}
		}
		item := editorFile{Path: relative, Language: language, Size: cached.size, Digest: cached.digest}
		if document := model.DocByPath[relative]; document != nil {
			item.Title = document.Title
			item.DocumentURL = document.OutputPath
		} else if language == "markdown" {
			item.Title = cached.title
		}
		nextCache[relative] = cached
		files = append(files, item)
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	w.cache = nextCache
	sort.SliceStable(files, func(i, j int) bool { return naturalCompare(files[i].Path, files[j].Path) < 0 })
	hash := sha256.New()
	for _, item := range files {
		_, _ = io.WriteString(hash, item.Path)
		_, _ = hash.Write([]byte{0})
		_, _ = io.WriteString(hash, item.Digest)
		_, _ = hash.Write([]byte{'\n'})
	}
	return files, hex.EncodeToString(hash.Sum(nil)), nil
}

func validateEditorPath(value string) error {
	if value == "" || value == "." || path.IsAbs(value) || filepath.IsAbs(value) {
		return workspaceFailure("invalid_path", "path must be relative")
	}
	if strings.Contains(value, "\\") || strings.ContainsRune(value, 0) || strings.Contains(value, "%") {
		return workspaceFailure("invalid_path", "path contains forbidden characters")
	}
	if path.Clean(value) != value {
		return workspaceFailure("invalid_path", "path is not canonical")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return workspaceFailure("invalid_path", "path contains a forbidden segment")
		}
	}
	if _, supported := editorLanguage(value); !supported {
		return workspaceFailure("unsupported_extension", "file extension is not supported")
	}
	return nil
}

func (w *editorWorkspace) resolve(value string, allowMissingFinal bool) (string, os.FileInfo, error) {
	if err := validateEditorPath(value); err != nil {
		return "", nil, err
	}
	if filepath.Clean(w.root) == filepath.Clean(w.repositoryRoot) && value == projectChangelogFile {
		return "", nil, workspaceFailure("path_forbidden", "the repository-root CHANGELOG.md is available only as the portal changelog")
	}
	current := w.root
	parts := strings.Split(value, "/")
	for index, part := range parts {
		current = filepath.Join(current, filepath.FromSlash(part))
		relative := strings.Join(parts[:index+1], "/")
		if w.excluded(relative, part) {
			return "", nil, workspaceFailure("path_forbidden", "path is excluded from the workspace")
		}
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) && allowMissingFinal && index == len(parts)-1 {
				return current, nil, nil
			}
			return "", nil, workspaceFailure("file_not_found", "file not found: %s", value)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", nil, workspaceFailure("path_forbidden", "symbolic links are forbidden in workspace paths")
		}
		if index < len(parts)-1 && !info.IsDir() {
			return "", nil, workspaceFailure("invalid_path", "path component is not a directory")
		}
		if index == len(parts)-1 {
			if !info.Mode().IsRegular() {
				return "", nil, workspaceFailure("path_forbidden", "workspace entry must be a regular file")
			}
			return current, info, nil
		}
	}
	return "", nil, workspaceFailure("invalid_path", "invalid path")
}

func (w *editorWorkspace) validateCreateDirectory(relative string) error {
	if relative == "" || relative == "." || path.Clean(relative) != relative {
		return workspaceFailure("invalid_path", "creation directory is not canonical")
	}
	current := w.root
	parts := strings.Split(relative, "/")
	for index, part := range parts {
		current = filepath.Join(current, filepath.FromSlash(part))
		partial := strings.Join(parts[:index+1], "/")
		if w.excluded(partial, part) {
			return workspaceFailure("path_forbidden", "directory is excluded from the workspace")
		}
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return workspaceFailure("path_forbidden", "creation directory contains a symbolic link or is not a directory")
		}
	}
	return nil
}

func (w *editorWorkspace) read(filePath string, model *Model, diagnostics bool) (editorFile, error) {
	absolute, _, err := w.resolve(filePath, false)
	if err != nil {
		return editorFile{}, err
	}
	content, err := os.ReadFile(absolute)
	if err != nil {
		return editorFile{}, err
	}
	language, _ := editorLanguage(filePath)
	item := editorFile{Path: filePath, Language: language, Size: int64(len(content)), Digest: contentDigest(content), Content: string(content)}
	if document := model.DocByPath[filePath]; document != nil {
		item.Title = document.Title
		item.DocumentURL = document.OutputPath
	} else if language == "markdown" {
		item.Title = analyzeMarkdown(string(content)).Title
	}
	if diagnostics {
		item.Diagnostics, err = w.diagnostics(filePath, content)
		if item.Diagnostics == nil {
			item.Diagnostics = []editorDiagnostic{}
		}
	}
	return item, err
}

func issueDiagnostics(issues []Issue) []editorDiagnostic {
	out := make([]editorDiagnostic, 0, len(issues))
	for _, issue := range issues {
		column := issue.Column
		if issue.Line > 0 && column == 0 {
			column = 1
		}
		out = append(out, editorDiagnostic{Severity: issue.Severity, Code: issue.Code, Message: issue.Message, Path: issue.DocumentPath, Line: issue.Line, Column: column})
	}
	return out
}

func jsonSyntaxDiagnostic(filePath string, content []byte) []editorDiagnostic {
	decoder := json.NewDecoder(bytes.NewReader(content))
	var value any
	err := decoder.Decode(&value)
	if err == nil {
		var trailing any
		if trailingErr := decoder.Decode(&trailing); trailingErr != io.EOF {
			err = trailingErr
		}
	}
	if err == nil {
		return nil
	}
	offset := int64(1)
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		offset = syntax.Offset
	}
	if offset < 1 {
		offset = 1
	}
	prefix := content
	if offset-1 < int64(len(content)) {
		prefix = content[:offset-1]
	}
	line := 1 + bytes.Count(prefix, []byte{'\n'})
	lastNewline := bytes.LastIndexByte(prefix, '\n')
	column := len(prefix) + 1
	if lastNewline >= 0 {
		column = len(prefix) - lastNewline
	}
	return []editorDiagnostic{{Severity: "error", Code: "invalid-json", Message: "Invalid JSON: " + err.Error(), Path: filePath, Line: line, Column: column}}
}

func (w *editorWorkspace) diagnostics(filePath string, content []byte) ([]editorDiagnostic, error) {
	language, _ := editorLanguage(filePath)
	if language == "yaml" {
		if isOpenAPIContractPath(filePath) {
			return issueDiagnostics(validateOpenAPIContract(filePath, content)), nil
		}
		return []editorDiagnostic{}, nil
	}
	if language == "json" {
		if isOpenAPIContractPath(filePath) {
			return issueDiagnostics(validateOpenAPIContract(filePath, content)), nil
		}
		syntax := jsonSyntaxDiagnostic(filePath, content)
		if len(syntax) > 0 || filePath != "screens/hotspots.json" {
			return syntax, nil
		}
		model, err := buildDocumentationModel(w.modelOptions, map[string][]byte{filePath: content})
		if err != nil {
			return nil, err
		}
		return issueDiagnostics(model.Issues), nil
	}
	model, err := buildDocumentationModel(w.modelOptions, map[string][]byte{filePath: content})
	if err != nil {
		return nil, err
	}
	return issueDiagnostics(model.Issues), nil
}

type staleFileError struct {
	file editorFile
}

func (e *staleFileError) Error() string {
	return "file changed by another process"
}

func (w *editorWorkspace) save(filePath string, content []byte, expectedDigest string) (editorFile, error) {
	if len(content) > editorContentLimit {
		return editorFile{}, workspaceFailure("content_too_large", "content exceeds 2 MiB")
	}
	absolute, info, err := w.resolve(filePath, false)
	if err != nil {
		return editorFile{}, err
	}
	current, err := os.ReadFile(absolute)
	if err != nil {
		return editorFile{}, err
	}
	if digest := contentDigest(current); expectedDigest == "" || digest != expectedDigest {
		language, _ := editorLanguage(filePath)
		return editorFile{}, &staleFileError{file: editorFile{Path: filePath, Language: language, Size: int64(len(current)), Digest: digest, Content: string(current)}}
	}
	temporary, err := os.CreateTemp(filepath.Dir(absolute), ".toudocu-edit-*")
	if err != nil {
		return editorFile{}, err
	}
	temporaryPath := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err = temporary.Chmod(info.Mode().Perm()); err != nil {
		return editorFile{}, err
	}
	if _, err = temporary.Write(content); err != nil {
		return editorFile{}, err
	}
	if err = temporary.Sync(); err != nil {
		return editorFile{}, err
	}
	if err = temporary.Close(); err != nil {
		return editorFile{}, err
	}
	if _, _, err = w.resolve(filePath, false); err != nil {
		return editorFile{}, err
	}
	latest, err := os.ReadFile(absolute)
	if err != nil {
		return editorFile{}, err
	}
	if digest := contentDigest(latest); digest != expectedDigest {
		language, _ := editorLanguage(filePath)
		return editorFile{}, &staleFileError{file: editorFile{Path: filePath, Language: language, Size: int64(len(latest)), Digest: digest, Content: string(latest)}}
	}
	if err = replaceEditorFile(temporaryPath, absolute); err != nil {
		return editorFile{}, err
	}
	keep = true
	language, _ := editorLanguage(filePath)
	return editorFile{Path: filePath, Language: language, Size: int64(len(content)), Digest: contentDigest(content), Content: string(content)}, nil
}
