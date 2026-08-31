package toudocu

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var stableEntityIDRE = regexp.MustCompile(`\b(?:UC|FLOW|SC|TR|MOD|ADR|TASK|BUG|BR|INV|CONTRACT)-[A-Z0-9][A-Z0-9-]*\b`)
var sourceDiffHunkRE = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// BuildDocumentationChanges compares Git-backed documentation snapshots.
func BuildDocumentationChanges(options Options) (*ChangeSetReport, error) {
	g, err := openGitChangeSource(options.InputDirectory, options.ChangeRenameSimilarity)
	if err != nil {
		return nil, err
	}
	g.cacheStatus = true
	configRoot, err := resolveChangeConfigurationRoot(options.RepositoryRoot, g)
	if err != nil {
		return nil, err
	}
	config, _, configErr := loadSiteConfig(configRoot)
	if configErr != nil {
		return nil, configErr
	}
	if issue := documentationVersionIssue(config); issue != nil {
		return &ChangeSetReport{SchemaVersion: 1, Changes: []DocumentationChange{}, Diagnostics: []Issue{*issue}}, nil
	}
	if options.ChangeBase == "" && options.ChangeBranchBase == "" {
		options.ChangeBase = config.Changes.DefaultBaseRef
	}
	if options.ChangeRenameSimilarity == 0 {
		options.ChangeRenameSimilarity = config.Changes.RenameSimilarity
		g.similarity = config.Changes.RenameSimilarity
	}
	if options.ChangeMaxSourceDiffBytes == 0 {
		options.ChangeMaxSourceDiffBytes = config.Changes.MaxSourceDiffBytes
	}
	if options.ChangeMaxRenderedFileBytes == 0 {
		options.ChangeMaxRenderedFileBytes = config.Changes.MaxRenderedFileBytes
	}
	if options.ChangeTranslationInput {
		options.ChangeIncludeTaskArtifacts = true
		options.ChangeIncludeAssets = true
		options.ChangeExclude = []string{
			path.Join(g.docsRel, "generated/**"),
			path.Join(g.docsRel, "cache/**"),
		}
	} else {
		options.ChangeIncludeTaskArtifacts = config.Changes.IncludeTaskArtifacts
		if !options.ChangeForceIncludeAssets {
			options.ChangeIncludeAssets = config.Changes.IncludeAssets
		} else {
			options.ChangeIncludeAssets = true
		}
		options.ChangeExclude = changeExcludesFromConfigurationRoot(g.root, configRoot, config.Changes.Exclude)
	}
	options.ChangeSemanticDiff, options.ChangeRenderedDiff = config.Changes.SemanticDiff, config.Changes.RenderedDiff
	baseRef := strings.TrimSpace(options.ChangeBase)
	if baseRef == "" {
		baseRef = "HEAD"
	}
	baseCommit := ""
	if baseRef != "index" {
		baseCommit, err = g.resolveCommit(baseRef)
		if err != nil {
			return nil, err
		}
	}
	if options.ChangeBranchBase != "" {
		branchCommit, err := g.resolveCommit(options.ChangeBranchBase)
		if err != nil {
			return nil, err
		}
		headCommit, err := g.resolveCommit("HEAD")
		if err != nil {
			return nil, err
		}
		mergeBase, err := g.run("merge-base", branchCommit, headCommit)
		if err != nil {
			return nil, &changeFailure{Code: 2, IssueCode: "git-merge-base-not-found", Err: fmt.Errorf("merge-base for %q and HEAD not found", options.ChangeBranchBase)}
		}
		baseCommit, baseRef = strings.TrimSpace(string(mergeBase)), "merge-base("+options.ChangeBranchBase+", HEAD)"
	}
	baseType := "commit"
	if baseRef == "index" {
		baseType = "index"
	}
	base := ChangeSide{Type: baseType, Revision: baseRef, Resolved: baseCommit, DisplayRef: baseRef}
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
		if failure, ok := err.(*changeFailure); ok {
			failure.IssueCode = "git-target-not-found"
		}
	}
	if err != nil {
		return nil, err
	}
	repository, err := g.repositoryState()
	if err != nil {
		return nil, err
	}
	files, err := g.listChanges(base, target)
	if err != nil {
		return nil, &changeFailure{Code: 3, IssueCode: "git-command-failed", Err: err}
	}
	report := &ChangeSetReport{SchemaVersion: 1, Repository: repository, Comparison: ChangeComparison{Base: base, Target: target}, Summary: ChangeSummary{Entities: map[string]int{}, Classifications: map[string]int{}}, Changes: []DocumentationChange{}, Diagnostics: []Issue{}}
	for _, file := range files {
		if requested := filepath.ToSlash(strings.TrimSpace(options.ChangeFile)); requested != "" && requested != file.path && requested != file.oldPath {
			continue
		}
		classification := classifyChangePath(g.docsRel, file.path)
		if (!options.ChangeIncludeTaskArtifacts && classification == "work-artifact") || (!options.ChangeIncludeAssets && classification == "asset") || excludedChangePath(file.path, options.ChangeExclude) {
			continue
		}
		change := buildDocumentationChange(g, base, target, file, options)
		report.Changes = append(report.Changes, change)
		addChangeSummary(&report.Summary, change)
	}
	coalesceEntityRenames(report)
	for i := range report.Changes {
		report.Changes[i].oldContent = nil
		report.Changes[i].newContent = nil
	}
	if options.ChangeTaskID != "" {
		taskContext, contextErr := buildChangeTaskContext(g, target, options.ChangeTaskID, options.ChangeTaskTree)
		if contextErr != nil {
			return nil, contextErr
		}
		report.taskContext = taskContext
		report.TaskImpact = buildTaskImpact(report, options.ChangeTaskID)
	}
	report.ChangeSetDigest = digestChangeSet(report)
	return report, nil
}

func resolveChangeConfigurationRoot(repositoryRoot string, g *gitChangeSource) (string, error) {
	if strings.TrimSpace(repositoryRoot) == "" {
		return g.root, nil
	}
	root, err := resolvePathForSafety(repositoryRoot)
	if err != nil {
		return "", err
	}
	if !pathContains(g.root, root) || !pathContains(root, g.docsRoot) {
		return "", fmt.Errorf("repository root must be inside the Git root and contain the documentation root")
	}
	return root, nil
}

func changeExcludesFromConfigurationRoot(gitRoot, configRoot string, patterns []string) []string {
	rel, err := filepath.Rel(gitRoot, configRoot)
	if err != nil || rel == "." {
		return append([]string(nil), patterns...)
	}
	prefix := filepath.ToSlash(rel)
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		result = append(result, path.Join(prefix, filepath.ToSlash(pattern)))
	}
	return result
}

func excludedChangePath(filePath string, patterns []string) bool {
	filePath = filepath.ToSlash(filePath)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		prefix := strings.TrimSuffix(strings.TrimSuffix(pattern, "**"), "*")
		if prefix != "" && strings.HasPrefix(filePath, prefix) {
			return true
		}
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}
	return false
}

func buildDocumentationChange(g *gitChangeSource, base, target ChangeSide, file gitFileChange, options Options) DocumentationChange {
	change := DocumentationChange{Status: file.status, Path: file.path, OldPath: file.oldPath, GitState: file.state, Classification: classifyChangePath(g.docsRel, file.path), EntitiesBefore: []ChangeEntity{}, EntitiesAfter: []ChangeEntity{}, SourceDiffHunks: []SourceDiffHunk{}, RenderedSections: []RenderedSectionChange{}, MermaidBlocks: []MermaidBlockChange{}, SemanticChanges: []SemanticChange{}, RelationChanges: []RelationChange{}, Diagnostics: []Issue{}}
	oldPath := file.path
	if file.oldPath != "" {
		oldPath = file.oldPath
	}
	var oldContent, newContent []byte
	var oldErr, newErr error
	if file.status != "added" && file.status != "untracked" {
		oldContent, oldErr = g.content(base, oldPath)
	}
	if file.status != "deleted" {
		newContent, newErr = g.content(target, file.path)
	}
	if oldErr != nil && !os.IsNotExist(oldErr) {
		change.Diagnostics = append(change.Diagnostics, Issue{Severity: "warning", Code: "change-old-version-missing", Message: oldErr.Error(), DocumentPath: file.oldPath})
	}
	if newErr != nil && !os.IsNotExist(newErr) {
		change.Diagnostics = append(change.Diagnostics, Issue{Severity: "warning", Code: "change-new-version-missing", Message: newErr.Error(), DocumentPath: file.path})
	}
	change.OldSize, change.NewSize = int64(len(oldContent)), int64(len(newContent))
	change.oldContent, change.newContent = oldContent, newContent
	change.Binary = isBinaryContent(oldContent) || isBinaryContent(newContent)
	if change.Classification == "asset" {
		change.Asset = buildAssetDiffMetadata(oldContent, newContent, file.status)
	}
	maxBytes := options.ChangeMaxSourceDiffBytes
	if maxBytes <= 0 {
		maxBytes = 2 * 1024 * 1024
	}
	if change.Binary {
		change.Diagnostics = append(change.Diagnostics, Issue{Severity: "info", Code: "git-binary-diff-unavailable", Message: "Text diff is unavailable for a binary file.", DocumentPath: file.path})
	} else if len(oldContent)+len(newContent) > maxBytes {
		change.Diagnostics = append(change.Diagnostics, Issue{Severity: "warning", Code: "change-file-too-large", Message: "Full diff is disabled by the size limit.", DocumentPath: file.path})
	} else if patch, err := g.diff(base, target, file); err == nil {
		change.SourceDiffAvailable = true
		change.Lines = countPatchLines(string(patch))
		if !options.ChangeOmitSourceDiff {
			change.SourceDiff = string(patch)
			change.SourceDiffHunks = parseSourceDiffHunks(change.SourceDiff)
		}
	} else {
		change.Diagnostics = append(change.Diagnostics, Issue{Severity: "warning", Code: "git-command-failed", Message: err.Error(), DocumentPath: file.path})
	}
	ext := strings.ToLower(filepath.Ext(file.path))
	if ext == ".md" && !change.Binary {
		change.RenderedDiffAvailable = options.ChangeRenderedDiff && len(oldContent)+len(newContent) <= options.ChangeMaxRenderedFileBytes
		change.Diagnostics = append(change.Diagnostics, markdownChangePolicyDiagnostics(oldContent, oldPath, "Old version")...)
		change.Diagnostics = append(change.Diagnostics, markdownChangePolicyDiagnostics(newContent, file.path, "New version")...)
		change.EntitiesBefore = entitiesFromMarkdown(oldContent, oldPath)
		change.EntitiesAfter = entitiesFromMarkdown(newContent, file.path)
		oldSemanticValid := markdownSemanticValid(oldContent)
		newSemanticValid := markdownSemanticValid(newContent)
		if !oldSemanticValid {
			change.Diagnostics = append(change.Diagnostics, Issue{Severity: "warning", Code: "semantic-old-version-invalid", Message: "The old Markdown version is invalid for semantic diff; source diff remains available.", DocumentPath: oldPath})
		}
		if !newSemanticValid {
			change.Diagnostics = append(change.Diagnostics, Issue{Severity: "warning", Code: "semantic-new-version-invalid", Message: "The new Markdown version is invalid for semantic diff; source diff remains available.", DocumentPath: file.path})
		}
		if len(change.EntitiesBefore) > 0 && len(change.EntitiesAfter) > 0 && change.EntitiesBefore[0].ID != "" && change.EntitiesAfter[0].ID != "" && change.EntitiesBefore[0].ID != change.EntitiesAfter[0].ID {
			change.Diagnostics = append(change.Diagnostics, Issue{Severity: "info", Code: "possible-entity-id-change", Message: "A stable ID changed; the entities are distinct until a relationship is declared explicitly.", DocumentPath: file.path})
		}
		if firstEntityType(change.EntitiesBefore, change.EntitiesAfter) == "screen" {
			change.Screen = buildScreenDiffMetadata(oldContent, newContent, oldPath, file.path)
		}
		change.RenderedSections, change.Diagnostics = renderedSectionDiff(oldContent, newContent, oldPath, file.path, change.Diagnostics)
		change.MermaidBlocks, change.Diagnostics = mermaidBlockDiff(oldContent, newContent, oldPath, file.path, change.Diagnostics)
		change.SemanticChanges = semanticMarkdownDiff(oldContent, newContent, oldPath, file.path, change.EntitiesBefore, change.EntitiesAfter)
		change.RelationChanges = relationMarkdownDiff(oldContent, newContent, change.EntitiesBefore, change.EntitiesAfter)
		change.SemanticDiffAvailable = options.ChangeSemanticDiff && oldSemanticValid && newSemanticValid
		if !options.ChangeSemanticDiff {
			change.SemanticChanges, change.RelationChanges = []SemanticChange{}, []RelationChange{}
		} else if !change.SemanticDiffAvailable {
			change.SemanticChanges, change.RelationChanges = []SemanticChange{}, []RelationChange{}
		}
	} else if (ext == ".yaml" || ext == ".yml" || ext == ".json") && !change.Binary {
		semantic, diagnostics, available := openAPIDiff(oldContent, newContent, oldPath, file.path)
		change.SemanticChanges, change.SemanticDiffAvailable = semantic, available
		change.Diagnostics = append(change.Diagnostics, diagnostics...)
		if available {
			change.EntitiesBefore = []ChangeEntity{{ID: contractID(oldPath), Type: "contract", Title: oldPath}}
			change.EntitiesAfter = []ChangeEntity{{ID: contractID(file.path), Type: "contract", Title: file.path}}
		}
	}
	return change
}

func markdownChangePolicyDiagnostics(content []byte, path, side string) []Issue {
	if len(content) == 0 {
		return nil
	}
	parsed := analyzeMarkdownPath(string(content), path)
	result := []Issue{}
	for _, diagnostic := range parsed.Diagnostics {
		if diagnostic.Code != "forbidden-raw-html" && diagnostic.Code != "forbidden-front-matter" {
			continue
		}
		result = append(result, Issue{Severity: "error", Code: diagnostic.Code, Message: side + ": " + diagnostic.Message, DocumentPath: path, Line: diagnostic.Range.Start.Line, Column: diagnostic.Range.Start.Column})
	}
	return result
}

func markdownSemanticValid(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	parsed := analyzeMarkdown(string(content))
	if strings.TrimSpace(parsed.Title) == "" {
		return false
	}
	for _, diagnostic := range parsed.Diagnostics {
		if diagnostic.Code == "unclosed-fence" {
			return false
		}
	}
	return true
}

func firstEntityType(groups ...[]ChangeEntity) string {
	for _, group := range groups {
		if len(group) > 0 {
			return group[0].Type
		}
	}
	return ""
}

func parseSourceDiffHunks(patch string) []SourceDiffHunk {
	lines := strings.SplitAfter(patch, "\n")
	hunks := []SourceDiffHunk{}
	for i := 0; i < len(lines); {
		header := strings.TrimSuffix(lines[i], "\n")
		match := sourceDiffHunkRE.FindStringSubmatch(header)
		if match == nil {
			i++
			continue
		}
		end := i + 1
		for end < len(lines) && sourceDiffHunkRE.FindStringSubmatch(strings.TrimSuffix(lines[end], "\n")) == nil {
			end++
		}
		oldStart, _ := strconv.Atoi(match[1])
		newStart, _ := strconv.Atoi(match[3])
		oldLines, newLines := 1, 1
		if match[2] != "" {
			oldLines, _ = strconv.Atoi(match[2])
		}
		if match[4] != "" {
			newLines, _ = strconv.Atoi(match[4])
		}
		hunks = append(hunks, SourceDiffHunk{
			ID: fmt.Sprintf("hunk-%d-%d", oldStart, newStart), Header: header,
			OldStart: oldStart, OldLines: oldLines, NewStart: newStart, NewLines: newLines,
			Patch: strings.Join(lines[i:end], ""),
		})
		i = end
	}
	return hunks
}

func renderedSectionDiff(oldContent, newContent []byte, oldPath, newPath string, diagnostics []Issue) ([]RenderedSectionChange, []Issue) {
	oldParsed, newParsed := analyzeMarkdown(string(oldContent)), analyzeMarkdown(string(newContent))
	type indexedSection struct {
		Section
		index int
	}
	oldByID, newByID := map[string][]indexedSection{}, map[string][]indexedSection{}
	for i, section := range oldParsed.Sections {
		oldByID[section.ID] = append(oldByID[section.ID], indexedSection{Section: section, index: i})
	}
	for i, section := range newParsed.Sections {
		newByID[section.ID] = append(newByID[section.ID], indexedSection{Section: section, index: i})
	}
	ids := map[string]bool{}
	for id := range oldByID {
		ids[id] = true
	}
	for id := range newByID {
		ids[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	result := []RenderedSectionChange{}
	for _, id := range ordered {
		olds, news := oldByID[id], newByID[id]
		if len(olds) > 1 || len(news) > 1 {
			diagnostics = append(diagnostics, Issue{Severity: "warning", Code: "rendered-section-match-ambiguous", Message: "Section with anchor " + id + " cannot be matched unambiguously.", DocumentPath: newPath})
			continue
		}
		var old, newer indexedSection
		oldOK, newOK := len(olds) == 1, len(news) == 1
		if oldOK {
			old = olds[0]
		}
		if newOK {
			newer = news[0]
		}
		status := "unchanged-section"
		switch {
		case !oldOK:
			status = "added-section"
		case !newOK:
			status = "removed-section"
		case normalizeSemanticText(old.Markdown) != normalizeSemanticText(newer.Markdown) || old.Title != newer.Title:
			status = "modified-section"
		case old.index != newer.index:
			status = "moved-section"
		}
		result = append(result, RenderedSectionChange{ID: id, Status: status, TitleBefore: old.Title, TitleAfter: newer.Title, AnchorBefore: anchorIf(oldOK, old.ID), AnchorAfter: anchorIf(newOK, newer.ID), SourceBefore: sectionLocation(oldPath, old.Section, oldOK), SourceAfter: sectionLocation(newPath, newer.Section, newOK)})
	}
	return result, diagnostics
}

func anchorIf(ok bool, value string) string {
	if ok {
		return value
	}
	return ""
}

func classifyChangePath(docsRel, path string) string {
	rel := strings.TrimPrefix(path, strings.TrimSuffix(docsRel, "/")+"/")
	lower := strings.ToLower(rel)
	if strings.HasPrefix(lower, "work/") || strings.Contains(lower, "/notes/") || strings.HasSuffix(lower, "notes.md") {
		return "work-artifact"
	}
	if strings.HasPrefix(lower, "contracts/") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json") {
		return "contract"
	}
	switch strings.ToLower(filepath.Ext(lower)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".svg":
		return "asset"
	}
	return "permanent-documentation"
}

func entitiesFromMarkdown(content []byte, path string) []ChangeEntity {
	if len(content) == 0 {
		return []ChangeEntity{}
	}
	parsed := analyzeMarkdown(string(content))
	typ := ClassifyDocument(strings.TrimPrefix(filepath.ToSlash(path), "docs/"))
	id := parsed.Metadata["id"]
	return []ChangeEntity{{ID: id, Type: typ, Title: parsed.Title}}
}

func semanticMarkdownDiff(oldContent, newContent []byte, oldPath, newPath string, before, after []ChangeEntity) []SemanticChange {
	changes := []SemanticChange{}
	var entity ChangeEntity
	if len(after) > 0 {
		entity = after[0]
	} else if len(before) > 0 {
		entity = before[0]
	}
	if len(oldContent) == 0 && len(newContent) > 0 {
		return []SemanticChange{{Kind: "entity-added", Entity: entity, After: entity, Summary: "Added " + semanticEntityName(entity) + ".", SourceAfter: &ChangeLocation{Path: newPath, Line: 1}}}
	}
	if len(newContent) == 0 && len(oldContent) > 0 {
		return []SemanticChange{{Kind: "entity-removed", Entity: entity, Before: entity, Summary: "Removed " + semanticEntityName(entity) + ".", SourceBefore: &ChangeLocation{Path: oldPath, Line: 1}}}
	}
	oldParsed, newParsed := analyzeMarkdown(string(oldContent)), analyzeMarkdown(string(newContent))
	keys := map[string]bool{}
	for k := range oldParsed.Metadata {
		keys[k] = true
	}
	for k := range newParsed.Metadata {
		keys[k] = true
	}
	ordered := make([]string, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	for _, key := range ordered {
		if oldParsed.Metadata[key] != newParsed.Metadata[key] {
			kind := "field-changed"
			if oldParsed.Metadata[key] == "" {
				kind = "field-added"
			} else if newParsed.Metadata[key] == "" {
				kind = "field-removed"
			}
			if key == "status" {
				kind = "status-changed"
			}
			changes = append(changes, SemanticChange{Kind: kind, Entity: entity, Field: key, Before: emptyAsNil(oldParsed.Metadata[key]), After: emptyAsNil(newParsed.Metadata[key]), Summary: semanticFieldSummary(kind, key), SourceBefore: metadataLocation(oldPath, oldParsed, key), SourceAfter: metadataLocation(newPath, newParsed, key)})
		}
	}
	oldSections := map[string]Section{}
	for _, s := range oldParsed.Sections {
		if s.Kind != "" {
			oldSections[string(s.Kind)] = s
		}
	}
	newSections := map[string]Section{}
	for _, s := range newParsed.Sections {
		if s.Kind != "" {
			newSections[string(s.Kind)] = s
		}
	}
	sectionIDs := map[string]bool{}
	for k := range oldSections {
		sectionIDs[k] = true
	}
	for k := range newSections {
		sectionIDs[k] = true
	}
	ids := make([]string, 0, len(sectionIDs))
	for k := range sectionIDs {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	for _, id := range ids {
		o, ook := oldSections[id]
		n, nok := newSections[id]
		kind := typedSectionChangeKind(entity.Type, n.Kind, o.Kind, "changed")
		summary := "Changed section "
		if !ook {
			kind, summary = typedSectionChangeKind(entity.Type, n.Kind, "", "added"), "Added section "
		} else if !nok {
			kind, summary = typedSectionChangeKind(entity.Type, "", o.Kind, "removed"), "Removed section "
		} else if normalizeSemanticText(o.Markdown) == normalizeSemanticText(n.Markdown) {
			continue
		}
		title := n.Title
		if title == "" {
			title = o.Title
		}
		changes = append(changes, SemanticChange{Kind: kind, Entity: entity, Field: id, Before: semanticSectionValue(o, ook), After: semanticSectionValue(n, nok), Summary: summary + title + ".", SourceBefore: sectionLocation(oldPath, o, ook), SourceAfter: sectionLocation(newPath, n, nok)})
	}
	type tableContract struct {
		columns []string
		line    int
	}
	oldTables, newTables := map[string]tableContract{}, map[string]tableContract{}
	for _, table := range oldParsed.Tables {
		if table.Kind != "" {
			oldTables[table.Kind] = tableContract{append([]string{}, table.Columns...), table.Range.Start.Line}
		}
	}
	for _, table := range newParsed.Tables {
		if table.Kind != "" {
			newTables[table.Kind] = tableContract{append([]string{}, table.Columns...), table.Range.Start.Line}
		}
	}
	tableKinds := map[string]bool{}
	for kind := range oldTables {
		tableKinds[kind] = true
	}
	for kind := range newTables {
		tableKinds[kind] = true
	}
	orderedTableKinds := make([]string, 0, len(tableKinds))
	for kind := range tableKinds {
		orderedTableKinds = append(orderedTableKinds, kind)
	}
	sort.Strings(orderedTableKinds)
	for _, tableKind := range orderedTableKinds {
		oldTable, oldOK := oldTables[tableKind]
		newTable, newOK := newTables[tableKind]
		if oldOK && newOK && strings.Join(oldTable.columns, "\x00") == strings.Join(newTable.columns, "\x00") {
			continue
		}
		kind := "field-changed"
		if !oldOK {
			kind = "field-added"
		} else if !newOK {
			kind = "field-removed"
		}
		change := SemanticChange{Kind: kind, Entity: entity, Field: "table." + tableKind, Summary: semanticFieldSummary(kind, "table."+tableKind)}
		if oldOK {
			change.Before = oldTable.columns
			change.SourceBefore = &ChangeLocation{Path: oldPath, Line: oldTable.line}
		}
		if newOK {
			change.After = newTable.columns
			change.SourceAfter = &ChangeLocation{Path: newPath, Line: newTable.line}
		}
		changes = append(changes, change)
	}
	changes = append(changes, semanticStableSubjectDiff(oldParsed, newParsed, oldPath, newPath, entity)...)
	if entity.Type == "work" {
		changes = append(changes, semanticVerificationDiff(oldParsed, newParsed, oldPath, newPath, entity)...)
	}
	return changes
}

func semanticEntityName(entity ChangeEntity) string {
	if entity.ID != "" {
		return "document " + entity.ID
	}
	if entity.Title != "" {
		return "document " + entity.Title
	}
	return "document"
}

func semanticFieldSummary(kind, key string) string {
	switch kind {
	case "field-added":
		return "Added field " + key + "."
	case "field-removed":
		return "Removed field " + key + "."
	case "status-changed":
		return "Changed status."
	default:
		return "Changed field " + key + "."
	}
}

func metadataLocation(path string, parsed markdownAnalysis, key string) *ChangeLocation {
	if line := parsed.MetadataLocations[key]; line > 0 {
		return &ChangeLocation{Path: path, Line: line}
	}
	return nil
}

func typedSectionChangeKind(entityType string, newKind, oldKind SectionKind, operation string) string {
	kind := newKind
	if kind == "" {
		kind = oldKind
	}
	prefix := "section"
	switch {
	case kind == SectionKindBusinessRules || kind == SectionKindRules || kind == SectionKindInvariants:
		prefix = "rule"
	case entityType == "work" && kind == SectionKindVerification:
		prefix = "verification"
	case entityType == "screen" && kind == "transitions":
		prefix = "transition"
	}
	return prefix + "-" + operation
}

type semanticSubject struct {
	entity ChangeEntity
	value  string
	line   int
}

func semanticStableSubjects(parsed markdownAnalysis) map[string]semanticSubject {
	result := map[string]semanticSubject{}
	add := func(value string, line int) {
		ids := stableEntityIDRE.FindAllString(value, -1)
		for _, id := range ids {
			typ := entityTypeFromID(id)
			if typ != "business-rule" && typ != "invariant" && typ != "transition" {
				continue
			}
			normalized := normalizeSemanticText(value)
			if existing, exists := result[id]; !exists || len(normalized) > len(existing.value) {
				result[id] = semanticSubject{entity: ChangeEntity{ID: id, Type: typ}, value: normalized, line: line}
			}
		}
	}
	for _, heading := range parsed.Headings {
		add(heading.Title, heading.Line+1)
	}
	for _, item := range parsed.ListItems {
		add(item.Text, item.Range.Start.Line)
	}
	for _, table := range parsed.Tables {
		for _, row := range table.Rows {
			add(strings.Join(row.Cells, " | "), row.Range.Start.Line)
		}
	}
	return result
}

func semanticStableSubjectDiff(oldParsed, newParsed markdownAnalysis, oldPath, newPath string, entity ChangeEntity) []SemanticChange {
	oldSubjects, newSubjects := semanticStableSubjects(oldParsed), semanticStableSubjects(newParsed)
	ids := map[string]bool{}
	for id := range oldSubjects {
		ids[id] = true
	}
	for id := range newSubjects {
		ids[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	changes := []SemanticChange{}
	for _, id := range ordered {
		old, oldOK := oldSubjects[id]
		newer, newOK := newSubjects[id]
		if oldOK && newOK && old.value == newer.value {
			continue
		}
		operation := "changed"
		if !oldOK {
			operation = "added"
		}
		if !newOK {
			operation = "removed"
		}
		prefix := "field"
		subject := newer.entity
		if !newOK {
			subject = old.entity
		}
		if subject.Type == "business-rule" || subject.Type == "invariant" {
			prefix = "rule"
		}
		if subject.Type == "transition" {
			prefix = "transition"
		}
		changes = append(changes, SemanticChange{Kind: prefix + "-" + operation, Entity: entity, Subject: &subject, Field: id, Before: subjectValue(old, oldOK), After: subjectValue(newer, newOK), Summary: semanticSubjectSummary(prefix, operation, id), SourceBefore: subjectLocation(oldPath, old, oldOK), SourceAfter: subjectLocation(newPath, newer, newOK)})
	}
	return changes
}

func semanticSubjectSummary(prefix, operation, id string) string {
	labels := map[string]string{"rule": "rule", "transition": "transition", "field": "item"}
	actions := map[string]string{"added": "Added", "removed": "Removed", "changed": "Changed"}
	return actions[operation] + " " + labels[prefix] + " " + id + "."
}

func subjectValue(subject semanticSubject, exists bool) any {
	if !exists {
		return nil
	}
	return subject.value
}

func subjectLocation(path string, subject semanticSubject, exists bool) *ChangeLocation {
	if !exists {
		return nil
	}
	return &ChangeLocation{Path: path, Line: subject.line}
}

func semanticVerificationDiff(oldParsed, newParsed markdownAnalysis, oldPath, newPath string, entity ChangeEntity) []SemanticChange {
	type criterion struct {
		completed bool
		text      string
		line      int
	}
	collect := func(parsed markdownAnalysis) map[string]criterion {
		result := map[string]criterion{}
		for _, task := range parsed.Tasks {
			id := regexp.MustCompile(`\bAC-[A-Z0-9-]+\b`).FindString(task.Text)
			if id != "" {
				result[id] = criterion{completed: task.Completed, text: normalizeSemanticText(task.Text), line: task.Line + 1}
			}
		}
		return result
	}
	oldCriteria, newCriteria := collect(oldParsed), collect(newParsed)
	ids := map[string]bool{}
	for id := range oldCriteria {
		ids[id] = true
	}
	for id := range newCriteria {
		ids[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	changes := []SemanticChange{}
	for _, id := range ordered {
		old, oldOK := oldCriteria[id]
		newer, newOK := newCriteria[id]
		if oldOK && newOK && old.completed == newer.completed && old.text == newer.text {
			continue
		}
		before, after := any(nil), any(nil)
		if oldOK {
			before = map[string]any{"completed": old.completed, "text": old.text}
		}
		if newOK {
			after = map[string]any{"completed": newer.completed, "text": newer.text}
		}
		change := SemanticChange{Kind: "verification-changed", Entity: entity, Subject: &ChangeEntity{ID: id, Type: "acceptance-criterion"}, Field: "acceptanceCriteria", Before: before, After: after, Summary: "Changed criterion " + id + "."}
		if oldOK {
			change.SourceBefore = &ChangeLocation{Path: oldPath, Line: old.line}
		}
		if newOK {
			change.SourceAfter = &ChangeLocation{Path: newPath, Line: newer.line}
		}
		changes = append(changes, change)
	}
	return changes
}

func relationMarkdownDiff(oldContent, newContent []byte, before, after []ChangeEntity) []RelationChange {
	var source ChangeEntity
	if len(after) > 0 {
		source = after[0]
	} else if len(before) > 0 {
		source = before[0]
	}
	oldIDs, newIDs := map[string]bool{}, map[string]bool{}
	for _, id := range stableEntityIDRE.FindAllString(string(oldContent), -1) {
		if id != source.ID {
			oldIDs[id] = true
		}
	}
	for _, id := range stableEntityIDRE.FindAllString(string(newContent), -1) {
		if id != source.ID {
			newIDs[id] = true
		}
	}
	all := map[string]bool{}
	for id := range oldIDs {
		all[id] = true
	}
	for id := range newIDs {
		all[id] = true
	}
	ordered := make([]string, 0, len(all))
	for id := range all {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	changes := []RelationChange{}
	for _, id := range ordered {
		kind := ""
		if !oldIDs[id] && newIDs[id] {
			kind = "relation-added"
		}
		if oldIDs[id] && !newIDs[id] {
			kind = "relation-removed"
		}
		if kind != "" {
			changes = append(changes, RelationChange{Kind: kind, Source: source, Target: ChangeEntity{ID: id, Type: entityTypeFromID(id)}})
		}
	}
	return changes
}

func entityTypeFromID(id string) string {
	prefix := strings.SplitN(id, "-", 2)[0]
	return map[string]string{"UC": "use-case", "FLOW": "flow", "SC": "screen", "TR": "transition", "MOD": "module", "ADR": "decision", "TASK": "work", "BUG": "work", "BR": "business-rule", "INV": "invariant", "CONTRACT": "contract"}[prefix]
}

func normalizeSemanticText(value string) string { return strings.Join(strings.Fields(value), " ") }
func emptyAsNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func semanticSectionValue(section Section, exists bool) any {
	if !exists {
		return nil
	}
	return normalizeSemanticText(section.Markdown)
}
func sectionLocation(path string, section Section, exists bool) *ChangeLocation {
	if !exists {
		return nil
	}
	return &ChangeLocation{Path: path, Line: section.StartLine}
}

func countPatchLines(patch string) ChangeLineStats {
	stats := ChangeLineStats{}
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.Added++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.Deleted++
		}
	}
	return stats
}

func addChangeSummary(summary *ChangeSummary, change DocumentationChange) {
	switch change.Status {
	case "added":
		summary.Files.Added++
	case "untracked":
		summary.Files.Untracked++
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
	}
	summary.Lines.Added += change.Lines.Added
	summary.Lines.Deleted += change.Lines.Deleted
	summary.Classifications[change.Classification]++
	seen := map[string]bool{}
	for _, entity := range append(append([]ChangeEntity{}, change.EntitiesBefore...), change.EntitiesAfter...) {
		key := entity.Type + ":" + entity.ID
		if !seen[key] {
			summary.Entities[entity.Type]++
			seen[key] = true
		}
	}
}

func coalesceEntityRenames(report *ChangeSetReport) {
	removed := map[string]int{}
	added := map[string]int{}
	for i, change := range report.Changes {
		if len(change.EntitiesBefore) == 0 && len(change.EntitiesAfter) == 0 {
			continue
		}
		if change.Status == "deleted" && change.EntitiesBefore[0].ID != "" {
			removed[change.EntitiesBefore[0].ID] = i
		}
		if (change.Status == "added" || change.Status == "untracked") && len(change.EntitiesAfter) > 0 && change.EntitiesAfter[0].ID != "" {
			added[change.EntitiesAfter[0].ID] = i
		}
	}
	drop := map[int]bool{}
	for id, oldIndex := range removed {
		newIndex, ok := added[id]
		if !ok {
			continue
		}
		old, newer := report.Changes[oldIndex], report.Changes[newIndex]
		newer.Status, newer.OldPath, newer.EntitiesBefore = "renamed", old.Path, old.EntitiesBefore
		newer.SourceDiff = old.SourceDiff + newer.SourceDiff
		newer.SourceDiffHunks = parseSourceDiffHunks(newer.SourceDiff)
		newer.RenderedSections = append(old.RenderedSections, newer.RenderedSections...)
		if old.Asset != nil || newer.Asset != nil {
			merged := &AssetDiffMetadata{}
			if old.Asset != nil {
				merged.Before = old.Asset.Before
			}
			if newer.Asset != nil {
				merged.After = newer.Asset.After
			}
			newer.Asset = merged
		}
		if old.Screen != nil || newer.Screen != nil {
			merged := &ScreenDiffMetadata{Transitions: []ScreenTransitionChange{}}
			if old.Screen != nil {
				merged.Before = old.Screen.Before
			}
			if newer.Screen != nil {
				merged.After = newer.Screen.After
			}
			newer.Screen = buildScreenDiffFromSnapshots(merged.Before, merged.After, old.Screen, newer.Screen)
		}
		newer.Lines.Added += old.Lines.Added
		newer.Lines.Deleted += old.Lines.Deleted
		if old.SemanticDiffAvailable && newer.SemanticDiffAvailable {
			newer.SemanticChanges = semanticMarkdownDiff(old.oldContent, newer.newContent, old.Path, newer.Path, old.EntitiesBefore, newer.EntitiesAfter)
		} else {
			newer.SemanticChanges = []SemanticChange{}
		}
		newer.SemanticChanges = append([]SemanticChange{{Kind: "entity-moved", Entity: newer.EntitiesAfter[0], Before: old.Path, After: newer.Path, Summary: "Entity " + id + " moved."}}, newer.SemanticChanges...)
		report.Changes[newIndex] = newer
		drop[oldIndex] = true
	}
	if len(drop) == 0 {
		return
	}
	filtered := report.Changes[:0]
	for i, change := range report.Changes {
		if !drop[i] {
			filtered = append(filtered, change)
		}
	}
	report.Changes = filtered
	// Rebuild summary after entity-aware rename classification.
	report.Summary = ChangeSummary{Entities: map[string]int{}, Classifications: map[string]int{}}
	for _, change := range report.Changes {
		addChangeSummary(&report.Summary, change)
	}
}

func digestChangeSet(report *ChangeSetReport) string {
	copyReport := *report
	copyReport.ChangeSetDigest = ""
	data, _ := json.Marshal(copyReport)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func buildChangeTaskContext(g *gitChangeSource, target ChangeSide, taskID string, tree bool) (*changeTaskContext, error) {
	if tree && !strings.HasPrefix(taskID, "TASK-") {
		return nil, fmt.Errorf("--tree is available only for TASK-* work items")
	}
	filterID := taskID
	if tree {
		filterID = ""
	}
	tasks, err := g.taskDocuments(target, filterID)
	if err != nil {
		return nil, err
	}
	root, ok := tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	selected := []string{taskID}
	if tree {
		children := map[string][]string{}
		for id, task := range tasks {
			if !strings.HasPrefix(id, "TASK-") {
				continue
			}
			parent := strings.TrimSpace(analyzeMarkdown(string(task.content)).Metadata["parentTask"])
			if parent != "" {
				children[parent] = append(children[parent], id)
			}
		}
		seen := map[string]bool{taskID: true}
		var add func(string)
		add = func(id string) {
			sort.Slice(children[id], func(i, j int) bool { return naturalCompare(children[id][i], children[id][j]) < 0 })
			for _, child := range children[id] {
				if seen[child] {
					continue
				}
				seen[child] = true
				selected = append(selected, child)
				add(child)
			}
		}
		add(taskID)
	}
	context := &changeTaskContext{content: root.content, path: root.path, docsRel: g.docsRel, pathExists: map[string]bool{}, tasks: tasks, selected: selected}
	for _, id := range selected {
		task := tasks[id]
		for _, path := range declaredTaskDocumentation(task.content, task.path, g.docsRel) {
			if _, contentErr := g.content(target, path); contentErr == nil {
				context.pathExists[path] = true
			} else if !os.IsNotExist(contentErr) {
				return nil, contentErr
			}
		}
	}
	return context, nil
}

func buildTaskImpact(report *ChangeSetReport, taskID string) *TaskImpactReport {
	impact := &TaskImpactReport{TaskID: taskID, Declared: []TaskImpactEntry{}, Actual: []TaskImpactEntry{}, TaskChanges: []DocumentationChange{}, Diagnostics: []Issue{}}
	content, taskPath, docsRel := reportTaskDocumentContent(report, taskID)
	selectedPaths := map[string]bool{taskPath: true}
	declaredBy := map[string][]string{}
	scope := []string{}
	if report.taskContext != nil && len(report.taskContext.selected) > 0 {
		for _, id := range report.taskContext.selected {
			task := report.taskContext.tasks[id]
			selectedPaths[task.path] = true
			for _, path := range declaredTaskDocumentation(task.content, task.path, docsRel) {
				declaredBy[path] = append(declaredBy[path], id)
			}
			scope = append(scope, taskScopePaths(task.content)...)
		}
	} else {
		for _, path := range declaredTaskDocumentation(content, taskPath, docsRel) {
			declaredBy[path] = []string{taskID}
		}
		scope = taskScopePaths(content)
	}
	for _, change := range report.Changes {
		if selectedPaths[change.Path] || selectedPaths[change.OldPath] {
			impact.TaskChanges = append(impact.TaskChanges, change)
			continue
		}
		impact.Actual = append(impact.Actual, TaskImpactEntry{Path: change.Path, Changed: true, Created: change.Status == "added" || change.Status == "untracked"})
	}
	declared := make([]string, 0, len(declaredBy))
	for path := range declaredBy {
		declared = append(declared, path)
	}
	sort.Strings(declared)
	changed := map[string]DocumentationChange{}
	for _, change := range report.Changes {
		changed[change.Path] = change
	}
	for _, path := range declared {
		change, ok := changed[path]
		entry := TaskImpactEntry{Path: path, Declared: true, Changed: ok, DeclaredBy: uniqueStrings(declaredBy[path])}
		if ok {
			entry.Created = change.Status == "added" || change.Status == "untracked"
		}
		impact.Declared = append(impact.Declared, entry)
		if !ok {
			code, message := "declared-document-not-changed", path+" is declared by the task but was not changed."
			if !taskTargetPathExists(report, path) {
				code, message = "declared-document-not-created", path+" is declared as a new document but was not created."
			}
			impact.Diagnostics = append(impact.Diagnostics, Issue{Severity: "warning", Code: code, Message: message, DocumentPath: path})
		}
	}
	declaredSet := map[string]bool{}
	for _, path := range declared {
		declaredSet[path] = true
	}
	for _, entry := range impact.Actual {
		if !declaredSet[entry.Path] {
			code := "undeclared-document-change"
			if entry.Created {
				code = "undeclared-document-created"
			}
			impact.Diagnostics = append(impact.Diagnostics, Issue{Severity: "warning", Code: code, Message: entry.Path + " changed but is not declared by the task.", DocumentPath: entry.Path})
		}
		if len(scope) > 0 && !pathMatchesTaskScope(entry.Path, scope) {
			impact.Diagnostics = append(impact.Diagnostics, Issue{Severity: "warning", Code: "documentation-change-outside-task-scope", Message: entry.Path + " is outside task scope.", DocumentPath: entry.Path})
		}
	}
	if len(declared) == 0 {
		for _, change := range report.Changes {
			if change.Classification == "permanent-documentation" {
				impact.Diagnostics = append(impact.Diagnostics, Issue{Severity: "warning", Code: "missing-documentation-impact-entry", Message: "The task changes durable documentation without an explicit documentation impact entry.", DocumentPath: change.Path})
				break
			}
		}
	}
	return impact
}

var documentationPathRE = regexp.MustCompile(`(?:\.\.?/)*(?:[A-Za-z0-9_.-]+/)*[A-Za-z0-9_.-]+\.(?:md|ya?ml|json|png|jpe?g|webp|svg)`)

func normalizeTaskDocumentationPath(value, taskPath, docsRel string, relativeToTask bool) string {
	value = normalizeSlashes(strings.TrimSpace(value))
	if value == "" || strings.Contains(value, "://") || strings.HasPrefix(value, "#") || filepath.IsAbs(filepath.FromSlash(value)) {
		return ""
	}
	if pathPart, _, _ := splitLinkDestination(value); pathPart != "" {
		value = pathPart
	}
	var candidate string
	switch {
	case value == docsRel || strings.HasPrefix(value, strings.TrimSuffix(docsRel, "/")+"/"):
		candidate = value
	case relativeToTask || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../"):
		candidate = filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(taskPath)), filepath.FromSlash(value)))
	default:
		candidate = filepath.ToSlash(filepath.Join(filepath.FromSlash(docsRel), filepath.FromSlash(value)))
	}
	candidate = filepath.ToSlash(filepath.Clean(filepath.FromSlash(candidate)))
	docsPath := filepath.Clean(filepath.FromSlash(docsRel))
	relative, err := filepath.Rel(docsPath, filepath.Clean(filepath.FromSlash(candidate)))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return ""
	}
	return candidate
}

func declaredTaskDocumentation(content []byte, taskPath, docsRel string) []string {
	if len(content) == 0 {
		return []string{}
	}
	parsed := analyzeMarkdown(string(content))
	var impactSection *Section
	for _, section := range parsed.Sections {
		if section.Kind == SectionKindDocumentationImpact {
			copy := section
			impactSection = &copy
			break
		}
	}
	if impactSection == nil {
		return []string{}
	}
	seen := map[string]bool{}
	paths := []string{}
	links := []Link{}
	for _, link := range parsed.Links {
		if !link.Image && link.Line > impactSection.StartLine && link.Line <= impactSection.EndLine {
			links = append(links, link)
		}
	}
	add := func(value string, relativeToTask bool) {
		candidate := normalizeTaskDocumentationPath(value, taskPath, docsRel, relativeToTask)
		if candidate != "" && !seen[candidate] {
			seen[candidate] = true
			paths = append(paths, candidate)
		}
	}
	for _, candidate := range documentationPathRE.FindAllString(impactSection.Markdown, -1) {
		insideLink := false
		for _, link := range links {
			if strings.Contains(link.Destination, candidate) {
				insideLink = true
				break
			}
		}
		if insideLink {
			continue
		}
		add(candidate, strings.HasPrefix(candidate, "."))
	}
	for _, link := range links {
		add(link.Destination, true)
	}
	sort.Strings(paths)
	return paths
}

func taskDocumentContent(repositoryRoot, taskID string) (string, []byte) {
	workRoot := filepath.Join(repositoryRoot, "docs", "work")
	selectedPath := ""
	var content []byte
	_ = filepath.WalkDir(workRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			return nil
		}
		candidate, readErr := os.ReadFile(path)
		if readErr == nil && taskIDFromContent(candidate) == taskID {
			selectedPath = filepath.ToSlash(toPosixRelative(repositoryRoot, path))
			content = candidate
		}
		return nil
	})
	return selectedPath, content
}

func taskScopePaths(content []byte) []string {
	if len(content) == 0 {
		return nil
	}
	parsed := analyzeMarkdown(string(content))
	values := []string{}
	for _, section := range parsed.Sections {
		if section.Kind != SectionKindScope {
			continue
		}
		for _, line := range strings.Split(section.Markdown, "\n") {
			start := strings.Index(line, "`")
			if start < 0 {
				continue
			}
			end := strings.Index(line[start+1:], "`")
			if end < 0 {
				continue
			}
			value := filepath.ToSlash(strings.TrimSpace(line[start+1 : start+1+end]))
			if value != "" {
				values = append(values, value)
			}
		}
	}
	return values
}

func reportTaskDocumentContent(report *ChangeSetReport, taskID string) ([]byte, string, string) {
	if report.taskContext != nil {
		return report.taskContext.content, report.taskContext.path, report.taskContext.docsRel
	}
	// Keep hand-built reports usable in focused unit tests. Production reports
	// always have taskContext populated from their selected target snapshot.
	path, content := taskDocumentContent(report.Repository.Root, taskID)
	return content, path, "docs"
}

func taskTargetPathExists(report *ChangeSetReport, documentPath string) bool {
	if report.taskContext != nil {
		return report.taskContext.pathExists[documentPath]
	}
	_, err := os.Stat(filepath.Join(report.Repository.Root, filepath.FromSlash(documentPath)))
	return err == nil
}

func pathMatchesTaskScope(path string, scope []string) bool {
	for _, item := range scope {
		if item == "." || item == "./" {
			return true
		}
		if strings.HasSuffix(item, "/") && strings.HasPrefix(path, item) {
			return true
		}
		if item == path {
			return true
		}
		if matched, _ := filepath.Match(item, path); matched {
			return true
		}
	}
	return false
}
