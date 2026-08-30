package toudocu

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var archiveYearRE = regexp.MustCompile(`^[0-9]{4}$`)

// taskArchivePathInfo reports whether a path is inside the reserved task
// archive, the archive year when valid, and whether it follows the supported
// work/archive/YYYY/file.md layout.
func taskArchivePathInfo(sourcePath string) (archived bool, year string, valid bool) {
	normalized := strings.TrimPrefix(path.Clean(normalizeSlashes(sourcePath)), "./")
	parts := strings.Split(normalized, "/")
	if len(parts) < 2 || parts[0] != "work" || parts[1] != "archive" {
		return false, "", true
	}
	if len(parts) == 4 && archiveYearRE.MatchString(parts[2]) && strings.EqualFold(path.Ext(parts[3]), ".md") {
		return true, parts[2], true
	}
	return true, "", false
}

func taskMoveIssue(code, message, documentPath string, line int) Issue {
	return Issue{Severity: "error", Code: code, Message: message, DocumentPath: documentPath, Line: line}
}

func taskMoveReport(kind string) TaskMoveReport {
	return TaskMoveReport{
		SchemaVersion: 1,
		Kind:          kind,
		Generator:     GeneratorInfo{Name: "Toudocu", Version: Version},
		Status:        "blocked",
		Issues:        []Issue{},
	}
}

func taskMoveResolutionKey(resolved ResolvedLink) string {
	target := ""
	if resolved.TargetDocument != nil {
		target = resolved.TargetDocument.SourcePath
	}
	return strings.Join([]string{
		target,
		resolved.RepositoryPath,
		resolved.RepositoryKind,
		resolved.AssetPath,
		resolved.GeneratedTarget,
		fmt.Sprint(resolved.External),
		fmt.Sprint(resolved.Blocked),
		fmt.Sprint(resolved.Broken),
		fmt.Sprint(resolved.RepositoryEscape),
		fmt.Sprint(resolved.RepositoryAsset),
		fmt.Sprint(resolved.ActiveAsset),
		fmt.Sprint(resolved.UnsafeImage),
	}, "\x00")
}

func taskMoveLinkIssues(model *Model, document *Document, destinationPath, destinationAbsolute string) []Issue {
	issues := []Issue{}
	for _, source := range model.Documents {
		if source == document {
			continue
		}
		for _, resolved := range source.ResolvedLinks {
			if resolved.TargetDocument == document {
				issues = append(issues, taskMoveIssue(
					"task-move-incoming-link",
					fmt.Sprintf("Link %q points to the file being moved; update or remove it before moving the task.", resolved.Destination),
					source.SourcePath,
					resolved.Line+1,
				))
			}
		}
	}

	alternate := *document
	alternate.SourcePath = destinationPath
	alternate.AbsolutePath = destinationAbsolute
	alternate.OutputPath = outputPathForDocument(destinationPath)
	for _, resolved := range document.ResolvedLinks {
		pathPart, _, hash := splitLinkDestination(resolved.Destination)
		if resolved.External || (pathPart == "" && hash != "") {
			continue
		}
		after := resolveLocalLink(model, &alternate, resolved.Link)
		if taskMoveResolutionKey(resolved) == taskMoveResolutionKey(after) {
			continue
		}
		issues = append(issues, taskMoveIssue(
			"task-move-outgoing-link",
			fmt.Sprintf("Link %q would resolve differently after the move.", resolved.Destination),
			document.SourcePath,
			resolved.Line+1,
		))
	}
	return issues
}

func validateTaskMovePaths(model *Model, source, destination string) error {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.Mode().IsRegular() {
		return fmt.Errorf("source task must be a regular file")
	}
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("destination file already exists: %s", toPosixRelative(model.RootDirectory, destination))
	} else if !os.IsNotExist(err) {
		return err
	}
	resolvedRoot, err := resolvePathForSafety(model.RootDirectory)
	if err != nil {
		return err
	}
	resolvedSource, err := resolvePathForSafety(source)
	if err != nil {
		return err
	}
	resolvedDestination, err := resolvePathForSafety(destination)
	if err != nil {
		return err
	}
	if !ensureInside(resolvedRoot, resolvedSource) || !ensureInside(resolvedRoot, resolvedDestination) {
		return fmt.Errorf("move path escapes the documentation directory")
	}
	return nil
}

func moveFileNoReplace(source, destination string) error {
	if err := os.Link(source, destination); err != nil {
		return fmt.Errorf("could not reserve destination file without overwriting: %w", err)
	}
	if err := os.Remove(source); err != nil {
		if cleanupErr := os.Remove(destination); cleanupErr != nil {
			return fmt.Errorf("could not remove source file: %v; could not roll back destination: %w", err, cleanupErr)
		}
		return fmt.Errorf("could not remove source file: %w", err)
	}
	return nil
}

// MoveTask archives or restores one task without editing its Markdown.
func MoveTask(model *Model, options Options, operation string) (TaskMoveReport, error) {
	kind := "task-" + operation
	report := taskMoveReport(kind)
	report.Task.ID = options.TaskID
	item, err := findWorkItem(model, options.TaskID)
	if err != nil {
		report.Issues = append(report.Issues, taskMoveIssue("task-move-target", err.Error(), "", 0))
		return report, nil
	}
	report.Task = TaskMoveTask{ID: item.ID, Title: item.Title, Status: item.Status, Type: item.Type}
	report.SourcePath = item.Document
	document := model.DocByPath[item.Document]
	if document == nil {
		report.Issues = append(report.Issues, taskMoveIssue("task-move-source", "Task file not found in the model.", item.Document, 0))
		return report, nil
	}
	archived, archiveYear, archivePathValid := taskArchivePathInfo(item.Document)
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	switch operation {
	case "archive":
		if archived {
			report.Issues = append(report.Issues, taskMoveIssue("task-already-archived", "The task is already archived.", item.Document, item.line))
			return report, nil
		}
		if item.statusName != "done" && item.statusName != "cancelled" {
			report.Issues = append(report.Issues, taskMoveIssue("task-not-terminal", "Only a Done or Cancelled task can be archived.", item.Document, item.line))
			return report, nil
		}
		archiveYear = now.Format("2006")
		report.ArchiveYear = archiveYear
		report.DestinationPath = path.Join("work", "archive", archiveYear, path.Base(item.Document))
	case "restore":
		if !archived {
			report.Issues = append(report.Issues, taskMoveIssue("task-not-archived", "The task is not archived.", item.Document, item.line))
			return report, nil
		}
		if !archivePathValid {
			report.Issues = append(report.Issues, taskMoveIssue("invalid-task-archive-path", "The task is outside work/archive/YYYY/*.md.", item.Document, item.line))
			return report, nil
		}
		report.ArchiveYear = archiveYear
		report.DestinationPath = path.Join("work", path.Base(item.Document))
	default:
		return report, fmt.Errorf("unknown task move operation: %s", operation)
	}

	if operation == "archive" {
		report.Issues = append(report.Issues, document.Errors...)
	}
	if len(report.Issues) > 0 {
		return report, nil
	}
	source := document.AbsolutePath
	destination := filepath.Join(model.RootDirectory, filepath.FromSlash(report.DestinationPath))
	if err := validateTaskMovePaths(model, source, destination); err != nil {
		report.Issues = append(report.Issues, taskMoveIssue("unsafe-task-move", err.Error(), item.Document, item.line))
		return report, nil
	}
	report.Issues = append(report.Issues, taskMoveLinkIssues(model, document, report.DestinationPath, destination)...)
	if len(report.Issues) > 0 {
		return report, nil
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return report, err
	}
	if err := moveFileNoReplace(source, destination); err != nil {
		return report, err
	}
	if operation == "archive" {
		report.Status = "archived"
	} else {
		report.Status = "restored"
	}
	return report, nil
}

func printTaskMoveText(w io.Writer, report TaskMoveReport) {
	if report.Status == "archived" {
		_, _ = fmt.Fprintf(w, "Task %s archived: %s\n", report.Task.ID, report.DestinationPath)
		return
	}
	if report.Status == "restored" {
		_, _ = fmt.Fprintf(w, "Task %s restored: %s\n", report.Task.ID, report.DestinationPath)
		return
	}
	_, _ = fmt.Fprintf(w, "Task %s was not moved.\n", report.Task.ID)
	for _, issue := range report.Issues {
		location := issue.DocumentPath
		if issue.Line > 0 {
			location += fmt.Sprintf(":%d", issue.Line)
		}
		_, _ = fmt.Fprintf(w, "[ERROR] %s %s — %s\n", issue.Code, location, issue.Message)
	}
}
