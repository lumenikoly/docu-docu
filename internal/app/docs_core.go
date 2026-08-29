package toudocu

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var defaultExcludes = []string{".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", "coverage"}

var rootOrder = map[string]int{
	"index.md": 0, "status.md": 1, "roadmap.md": 2, "risks.md": 3,
	"ideas.md": 4, "notes.md": 5, "glossary.md": 6,
}

type statusGroup struct {
	Kind   string
	Symbol string
}

var statusGroups = map[string]statusGroup{
	"draft":           {Kind: "not-started", Symbol: "-"},
	"ready":           {Kind: "planned", Symbol: "~"},
	"planned":         {Kind: "planned", Symbol: "~"},
	"proposed":        {Kind: "planned", Symbol: "~"},
	"in-progress":     {Kind: "in-progress", Symbol: "="},
	"active":          {Kind: "in-progress", Symbol: "="},
	"blocked":         {Kind: "blocked", Symbol: "!"},
	"paused":          {Kind: "paused", Symbol: "="},
	"done":            {Kind: "done", Symbol: "+"},
	"open":            {Kind: "open", Symbol: "-"},
	"accepted":        {Kind: "accepted", Symbol: "+"},
	"rejected":        {Kind: "rejected", Symbol: "x"},
	"cancelled":       {Kind: "cancelled", Symbol: "x"},
	"superseded":      {Kind: "superseded", Symbol: ">"},
	"obsolete":        {Kind: "obsolete", Symbol: "x"},
	"review-required": {Kind: "review-required", Symbol: "!"},
	"risk-accepted":   {Kind: "risk-accepted", Symbol: "="},
}

func ClassifyDocument(relativePath string) string {
	normalized := strings.ToLower(normalizeSlashes(relativePath))
	base := path.Base(normalized)
	section := sectionTypeForPath(normalized)
	switch normalized {
	case "index.md":
		return "overview"
	case "status.md":
		return "status"
	case "roadmap.md":
		return "roadmap"
	case "risks.md":
		return "risks"
	case "notes.md":
		return "notes"
	case "ideas.md":
		return "ideas"
	}
	switch section {
	case SectionUseCases:
		return "use-case"
	case SectionModules:
		return "module"
	case SectionArchitecture:
		return "architecture"
	case SectionContracts:
		return "contract"
	case SectionDecisions:
		return "decision"
	case SectionFlows:
		return "flow"
	case SectionGuides:
		return "guide"
	case SectionDrafts:
		return "draft"
	case SectionReference:
		return "reference"
	case SectionQuality:
		if base == "index.md" {
			return "quality-index"
		}
		if strings.HasPrefix(base, "std-") {
			return "standard"
		}
	case SectionRunbooks:
		if base == "index.md" {
			return "runbook-index"
		}
		if strings.HasPrefix(base, "rb-") {
			return "runbook"
		}
	case SectionScreens:
		if base == "map.md" {
			return "screen-map"
		}
		if base == "index.md" {
			return "screen-index"
		}
		if strings.HasPrefix(base, "sc-") {
			return "screen"
		}
	case SectionWork:
		if strings.HasPrefix(base, "task-") || strings.HasPrefix(base, "bug-") {
			return "work"
		}
	}
	if base == "index.md" {
		return "document"
	}
	return "document"
}

const (
	projectChangelogFile   = "CHANGELOG.md"
	projectChangelogOutput = "project-changelog.html"
)

func loadProjectChangelog(repositoryRoot string, staleDays int, now time.Time, fallbackTitle string) (*Document, *Issue) {
	filePath := filepath.Join(repositoryRoot, projectChangelogFile)
	info, err := os.Lstat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, ptrIssue(newIssue("warning", "project-changelog-unavailable", "Could not inspect the repository-root CHANGELOG.md: "+err.Error(), projectChangelogFile, 0))
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, ptrIssue(newIssue("warning", "project-changelog-unavailable", "The repository-root CHANGELOG.md must be a regular file; the changelog page is hidden.", projectChangelogFile, 0))
	}
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ptrIssue(newIssue("warning", "project-changelog-unavailable", "Could not read the repository-root CHANGELOG.md: "+err.Error(), projectChangelogFile, 0))
	}
	content := string(contentBytes)
	parsed := analyzeMarkdown(content)
	title := parsed.Title
	if title == "" {
		title = fallbackTitle
	}
	updatedAt := info.ModTime().UTC()
	if value := parsed.Metadata["updated"]; value != "" {
		if parsedDate, ok := parseISODate(value); ok {
			updatedAt = parsedDate
		}
	} else if value := parsed.Metadata["date"]; value != "" {
		if parsedDate, ok := parseISODate(value); ok {
			updatedAt = parsedDate
		}
	}
	completed := 0
	for _, task := range parsed.Tasks {
		if task.Completed {
			completed++
		}
	}
	return &Document{
		ID: projectChangelogFile, AbsolutePath: filePath, SourcePath: projectChangelogFile,
		OutputPath: projectChangelogOutput, Directory: ".", FileName: projectChangelogFile,
		Type: "changelog", TypeLabel: localizedTypeLabel(nil, "changelog"), Title: title, Description: parsed.Description,
		Content: content, Headings: parsed.Headings,
		Sections: parsed.Sections, Metadata: parsed.Metadata, MetadataExtras: parsed.MetadataExtras, metadataLocations: parsed.MetadataLocations, metadataCounts: parsed.MetadataCounts, metadataBlocks: parsed.MetadataBlocks,
		Tasks:     parsed.Tasks,
		TaskStats: TaskStats{Total: len(parsed.Tasks), Completed: completed, Remaining: len(parsed.Tasks) - completed, Percent: progress(completed, len(parsed.Tasks))},
		Links:     parsed.Links, PlainText: parsed.PlainText, MTime: info.ModTime().UTC(), UpdatedAt: updatedAt,
		AgeDays: int(now.UTC().Sub(updatedAt).Hours() / 24), Stale: staleDays > 0 && int(now.UTC().Sub(updatedAt).Hours()/24) > staleDays,
		Status: StatusFor(""), markdownDiagnostics: markdownIssues(parsed, projectChangelogFile), mermaidBlocks: mermaidBlocksFromAnalysis(parsed), markdownTables: markdownTablesFromAnalysis(parsed),
	}, nil
}

func ptrIssue(issue Issue) *Issue { return &issue }

func projectChangelogFingerprint(repositoryRoot string) string {
	filePath := filepath.Join(repositoryRoot, projectChangelogFile)
	info, err := os.Lstat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		return "stat-error:" + err.Error()
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "unavailable:" + info.Mode().String()
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "read-error:" + err.Error()
	}
	return "content:" + contentDigest(content)
}

func outputPathForDocument(relativePath string) string {
	normalized := normalizeSlashes(relativePath)
	if len(normalized) >= 3 && strings.EqualFold(normalized[len(normalized)-3:], ".md") {
		return normalized[:len(normalized)-3] + ".html"
	}
	return normalized + ".html"
}

// StatusFor maps one canonical status to its presentation group.
func StatusFor(status string) StatusInfo {
	label := strings.TrimSpace(status)
	if label == "" {
		return StatusInfo{Kind: "neutral", Symbol: ".", Recognized: true}
	}
	if group, ok := statusGroups[label]; ok {
		return StatusInfo{Kind: group.Kind, Symbol: group.Symbol, Label: label, Recognized: true}
	}
	return StatusInfo{Kind: "neutral", Symbol: ".", Label: label}
}

func shouldExclude(relativePath, baseName string, excludes map[string]struct{}) bool {
	if strings.HasPrefix(baseName, ".") {
		return true
	}
	normalized := normalizeSlashes(relativePath)
	_, byName := excludes[baseName]
	_, byPath := excludes[normalized]
	return byName || byPath
}

type scannedFile struct {
	AbsolutePath string
	RelativePath string
}

func scanMarkdownFiles(root string, customExcludes []string, issues *[]Issue) []scannedFile {
	excludes := map[string]struct{}{}
	for _, value := range append(append([]string{}, defaultExcludes...), customExcludes...) {
		value = strings.TrimSpace(value)
		if value != "" {
			excludes[normalizeSlashes(value)] = struct{}{}
		}
	}
	files := []scannedFile{}
	var walk func(string)
	walk = func(directory string) {
		entries, err := os.ReadDir(directory)
		if err != nil {
			*issues = append(*issues, Issue{Severity: "error", Code: "directory-read-failed", Message: "Could not read directory: " + err.Error(), DocumentPath: toPosixRelative(root, directory)})
			return
		}
		sort.SliceStable(entries, func(i, j int) bool { return naturalCompare(entries[i].Name(), entries[j].Name()) < 0 })
		for _, entry := range entries {
			absolute := filepath.Join(directory, entry.Name())
			relative := toPosixRelative(root, absolute)
			info, err := os.Lstat(absolute)
			if err != nil {
				*issues = append(*issues, Issue{Severity: "warning", Code: "file-stat-failed", Message: "Could not inspect file: " + err.Error(), DocumentPath: relative})
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				if strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
					*issues = append(*issues, Issue{Severity: "warning", Code: "ignored-symlink", Message: "Markdown symbolic link ignored.", DocumentPath: relative})
				}
				continue
			}
			if info.IsDir() {
				if !shouldExclude(relative, entry.Name(), excludes) {
					walk(absolute)
				}
			} else if info.Mode().IsRegular() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				files = append(files, scannedFile{AbsolutePath: absolute, RelativePath: relative})
			}
		}
	}
	walk(root)
	return files
}

func newIssue(severity, code, message, documentPath string, line int) Issue {
	return Issue{Severity: severity, Code: code, Message: message, DocumentPath: documentPath, Line: line}
}

func documentLess(a, b *Document) bool {
	aRoot := !strings.Contains(a.SourcePath, "/")
	bRoot := !strings.Contains(b.SourcePath, "/")
	if aRoot && bRoot {
		aOrder, aKnown := rootOrder[strings.ToLower(a.SourcePath)]
		bOrder, bKnown := rootOrder[strings.ToLower(b.SourcePath)]
		if !aKnown {
			aOrder = 100
		}
		if !bKnown {
			bOrder = 100
		}
		if aOrder != bOrder {
			return aOrder < bOrder
		}
	}
	if aRoot != bRoot {
		return aRoot
	}
	return naturalCompare(a.SourcePath, b.SourcePath) < 0
}

func createDocument(file scannedFile, root string, staleDays int, now time.Time, issues *[]Issue, overlay map[string][]byte) *Document {
	contentBytes, exists := overlay[file.RelativePath]
	var err error
	if !exists {
		contentBytes, err = os.ReadFile(file.AbsolutePath)
	}
	if err != nil {
		*issues = append(*issues, newIssue("error", "file-read-failed", "Could not read file: "+err.Error(), file.RelativePath, 0))
		return nil
	}
	info, err := os.Stat(file.AbsolutePath)
	if err != nil {
		*issues = append(*issues, newIssue("error", "file-read-failed", "Could not inspect file: "+err.Error(), file.RelativePath, 0))
		return nil
	}
	content := string(contentBytes)
	parsed := analyzeMarkdown(content)
	typeName := ClassifyDocument(file.RelativePath)
	section := sectionTypeForPath(file.RelativePath)
	fallback := strings.TrimSuffix(path.Base(file.RelativePath), path.Ext(file.RelativePath))
	fallback = strings.ReplaceAll(strings.ReplaceAll(fallback, "-", " "), "_", " ")
	if fallback != "" {
		runes := []rune(fallback)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		fallback = string(runes)
	}
	updatedAt := info.ModTime().UTC()
	if value := parsed.Metadata["updated"]; value != "" {
		if parsedDate, ok := parseISODate(value); ok {
			updatedAt = parsedDate
		}
	} else if value := parsed.Metadata["date"]; value != "" {
		if parsedDate, ok := parseISODate(value); ok {
			updatedAt = parsedDate
		}
	}
	ageDays := int(now.UTC().Sub(updatedAt).Hours() / 24)
	completed := 0
	for _, task := range parsed.Tasks {
		if task.Completed {
			completed++
		}
	}
	title := parsed.Title
	if title == "" {
		title = fallback
	}
	directory := path.Dir(normalizeSlashes(file.RelativePath))
	return &Document{
		ID: file.RelativePath, AbsolutePath: file.AbsolutePath, SourcePath: normalizeSlashes(file.RelativePath),
		OutputPath: outputPathForDocument(file.RelativePath), Directory: directory, FileName: path.Base(file.RelativePath),
		Type: typeName, SectionType: section, TypeLabel: localizedTypeLabel(nil, typeName), Title: title, Description: parsed.Description,
		Content: content, Headings: parsed.Headings,
		Sections: parsed.Sections, Metadata: parsed.Metadata, MetadataExtras: parsed.MetadataExtras, metadataLocations: parsed.MetadataLocations, metadataCounts: parsed.MetadataCounts, metadataBlocks: parsed.MetadataBlocks,
		Tasks:     parsed.Tasks,
		TaskStats: TaskStats{Total: len(parsed.Tasks), Completed: completed, Remaining: len(parsed.Tasks) - completed, Percent: progress(completed, len(parsed.Tasks))},
		Links:     parsed.Links, PlainText: parsed.PlainText, MTime: info.ModTime().UTC(), UpdatedAt: updatedAt,
		AgeDays: ageDays, Stale: staleDays > 0 && ageDays > staleDays, Status: StatusFor(parsed.Metadata["status"]), markdownDiagnostics: markdownIssues(parsed, normalizeSlashes(file.RelativePath)), mermaidBlocks: mermaidBlocksFromAnalysis(parsed), markdownTables: markdownTablesFromAnalysis(parsed),
	}
}

func markdownIssues(parsed markdownAnalysis, sourcePath string) []Issue {
	issues := make([]Issue, 0, len(parsed.Diagnostics))
	for _, diagnostic := range parsed.Diagnostics {
		issues = append(issues, newIssue(diagnostic.Severity, diagnostic.Code, diagnostic.Message, sourcePath, diagnostic.Range.Start.Line))
	}
	return issues
}

func addDocumentIssue(model *Model, document *Document, issue Issue) {
	model.Issues = append(model.Issues, issue)
	if issue.Severity == "error" {
		document.Errors = append(document.Errors, issue)
	} else {
		document.Warnings = append(document.Warnings, issue)
	}
}

func hasSection(document *Document, kind SectionKind) bool {
	for _, section := range document.Sections {
		if section.Kind == kind {
			return true
		}
	}
	return false
}

func validateDocumentBasics(model *Model, document *Document) {
	for _, issue := range document.markdownDiagnostics {
		addDocumentIssue(model, document, issue)
	}
	validateSemanticAnnotations(model, document)
	if containsType([]string{"notes", "ideas"}, document.Type) {
		return
	}
	if strings.TrimSpace(document.Content) == "" {
		addDocumentIssue(model, document, newIssue("warning", "empty-document", "The document is empty.", document.SourcePath, 0))
	}
	hasH1 := false
	for _, heading := range document.Headings {
		if heading.Level == 1 {
			hasH1 = true
			break
		}
	}
	if !hasH1 {
		addDocumentIssue(model, document, newIssue("warning", "missing-h1", "The document has no level-one heading.", document.SourcePath, 0))
	}
	if document.Description == "" && containsType([]string{"overview", "module", "use-case", "architecture", "decision"}, document.Type) {
		addDocumentIssue(model, document, newIssue("warning", "missing-description", "The document has no introductory description.", document.SourcePath, 0))
	}
	if document.Stale {
		addDocumentIssue(model, document, newIssue("warning", "stale-document", fmt.Sprintf("The document has not been updated for %d days.", document.AgeDays), document.SourcePath, 0))
	}
	if document.Metadata["status"] != "" && !document.Status.Recognized {
		addDocumentIssue(model, document, newIssue("warning", "unknown-status", fmt.Sprintf("Unknown status %q.", document.Metadata["status"]), document.SourcePath, 0))
	}
	if containsType([]string{"status", "use-case", "module", "decision"}, document.Type) && document.Metadata["status"] == "" {
		addDocumentIssue(model, document, newIssue("warning", "missing-status", "The Status field is missing.", document.SourcePath, 0))
	}

	type sectionRule struct {
		Kind    SectionKind
		Message string
	}
	rules := map[string][]sectionRule{
		"use-case": {
			{SectionKindMainScenario, "The Main scenario section is missing."},
			{SectionKindPostconditions, "The Postconditions section is missing."},
			{SectionKindBusinessRules, "The Business rules section is missing."},
			{SectionKindImplementation, "The Implementation section is missing."},
		},
		"module": {
			{SectionKindCodeLocation, "The Code location section is missing."},
			{SectionKindBoundaries, "The Boundaries section is missing."},
			{SectionKindBusinessRules, "The Business rules section is missing."},
			{SectionKindInvariants, "The Invariants section is missing."},
			{SectionKindStableInterfaces, "The Stable interfaces section is missing."},
			{SectionKindRelatedUseCases, "The Related use cases section is missing."},
		},
		"decision": {
			{SectionKindContext, "The Context section is missing."},
			{SectionKindDecision, "The Decision section is missing."},
			{SectionKindConsequences, "The Consequences section is missing."},
		},
	}
	for _, rule := range rules[document.Type] {
		if !hasSection(document, rule.Kind) {
			addDocumentIssue(model, document, newIssue("warning", "missing-section", rule.Message, document.SourcePath, 0))
		}
	}
}

func containsType(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildDirectories(documents []*Document) map[string]struct{} {
	result := map[string]struct{}{}
	for _, document := range documents {
		directory := document.Directory
		for directory != "" && directory != "." {
			result[directory] = struct{}{}
			parent := path.Dir(directory)
			if parent == directory || parent == "." {
				break
			}
			directory = parent
		}
	}
	return result
}

func buildCollections(documents []*Document) map[string][]*Document {
	result := map[string][]*Document{}
	for _, document := range documents {
		result[document.Type] = append(result[document.Type], document)
	}
	for _, collection := range result {
		sort.SliceStable(collection, func(i, j int) bool { return documentLess(collection[i], collection[j]) })
	}
	return result
}

func validateGlobalStructure(model *Model) {
	if model.DocByPath["index.md"] == nil {
		model.Issues = append(model.Issues, newIssue("warning", "missing-index", "Required file index.md is missing.", "", 0))
	}
	if model.DocByPath["architecture/overview.md"] == nil {
		model.Issues = append(model.Issues, newIssue("error", "missing-architecture-overview", "Required file architecture/overview.md is missing.", "architecture/overview.md", 0))
	}
	titles := map[string][]*Document{}
	for _, document := range model.Documents {
		key := canonicalText(document.Title)
		if key != "" {
			titles[key] = append(titles[key], document)
		}
	}
	for _, group := range titles {
		if len(group) > 1 {
			for _, document := range group {
				addDocumentIssue(model, document, newIssue("warning", "duplicate-title", fmt.Sprintf("Heading %q is used by multiple documents.", document.Title), document.SourcePath, 0))
			}
		}
	}
}

func validateArchitectureDocuments(model *Model) {
	overview := model.DocByPath["architecture/overview.md"]
	listed := map[string]struct{}{}
	if overview != nil {
		for _, link := range overview.ResolvedLinks {
			if link.TargetDocument != nil && link.TargetDocument.Type == "architecture" && link.TargetDocument != overview && !link.Image {
				listed[link.TargetDocument.SourcePath] = struct{}{}
			}
		}
	}
	for _, document := range model.Collections["architecture"] {
		if document == overview {
			continue
		}
		if strings.TrimSpace(document.Metadata["architectureQuestion"]) == "" {
			addDocumentIssue(model, document, newIssue(
				"error",
				"missing-architecture-question",
				"A detailed architecture document must contain a non-empty Architecture question field.",
				document.SourcePath,
				0,
			))
		}
		if _, ok := listed[document.SourcePath]; !ok {
			addDocumentIssue(model, document, newIssue(
				"error",
				"unlisted-architecture-document",
				"A detailed architecture document must be linked directly from architecture/overview.md.",
				document.SourcePath,
				0,
			))
		}
	}
}

func assignUniqueOutputPaths(model *Model) {
	used := map[string]*Document{}
	if model.ProjectChangelog != nil {
		used[strings.ToLower(model.ProjectChangelog.OutputPath)] = model.ProjectChangelog
	}
	for _, document := range model.Documents {
		if id := strings.TrimSpace(document.Metadata["id"]); safeStableID(id) {
			switch document.Type {
			case "use-case":
				if strings.HasPrefix(id, "UC-") {
					document.OutputPath = path.Join("use-cases", id+".html")
				}
			case "flow":
				if strings.HasPrefix(id, "FLOW-") {
					document.OutputPath = path.Join("flows", id+".html")
				}
			case "screen":
				if strings.HasPrefix(id, "SC-") {
					document.OutputPath = path.Join("screens", id+".html")
				}
			}
		}
		candidate := document.OutputPath
		key := strings.ToLower(candidate)
		if previous := used[key]; previous != nil {
			base := strings.TrimSuffix(candidate, path.Ext(candidate))
			ext := path.Ext(candidate)
			for i := 2; ; i++ {
				candidate = fmt.Sprintf("%s-%d%s", base, i, ext)
				if used[strings.ToLower(candidate)] == nil {
					break
				}
			}
			document.OutputPath = candidate
			addDocumentIssue(model, document, newIssue("error", "output-path-collision", fmt.Sprintf("Output path conflicts with %s; assigned %s.", previous.SourcePath, candidate), document.SourcePath, 0))
		}
		used[strings.ToLower(document.OutputPath)] = document
	}
}

func safeStableID(id string) bool {
	if id == "" || strings.HasPrefix(id, "-") || strings.HasSuffix(id, "-") {
		return false
	}
	for _, value := range id {
		if value == '-' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' {
			continue
		}
		return false
	}
	return true
}

// BuildDocumentationModel reads, validates and cross-links all Markdown documents.
func BuildDocumentationModel(options Options) (*Model, error) {
	return buildDocumentationModel(options, nil)
}

func buildDocumentationModel(options Options, overlay map[string][]byte) (*Model, error) {
	root, err := filepath.Abs(options.InputDirectory)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("documentation directory not found: %s", root)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("the specified path is not a directory: %s", root)
	}
	repositoryRoot := options.RepositoryRoot
	if repositoryRoot == "" {
		repositoryRoot = filepath.Dir(root)
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return nil, err
	}
	siteConfig, brandingAssets, err := loadSiteConfig(repositoryRoot)
	if err != nil {
		return nil, err
	}
	localeRoots, err := selectLocaleProfile(&siteConfig, repositoryRoot, root)
	if err != nil {
		return nil, err
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	staleDays := options.StaleDays
	if staleDays < 0 {
		staleDays = 90
	}
	model := &Model{
		RootDirectory: root, RepositoryRoot: repositoryRoot, RepositoryURL: strings.TrimRight(options.RepositoryURL, "/"),
		RepositoryRef: options.RepositoryRef, GeneratedAt: now.UTC(), StaleDays: staleDays,
		DocByPath: map[string]*Document{}, Directories: map[string]struct{}{}, Assets: map[string]string{},
		BrandingAssets: brandingAssets, SiteConfig: siteConfig,
		Collections: map[string][]*Document{}, Knowledge: KnowledgeModel{},
		HealthOutputPath: "health.html", ReportOutputPath: "report.json", ScreenMapEnabled: true,
		sourceOverlay: overlay,
		strictPolicy:  options.Strict,
	}
	if model.RepositoryRef == "" {
		model.RepositoryRef = "main"
	}
	if issue := documentationVersionIssue(siteConfig); issue != nil {
		model.Issues = append(model.Issues, *issue)
		model.Stats = buildStats(model)
		return model, nil
	}
	files := scanMarkdownFiles(root, options.Excludes, &model.Issues)
	// A repository-root scan must not combine independently configured locales.
	if filepath.Clean(root) == filepath.Clean(repositoryRoot) && len(localeRoots) > 0 {
		filtered := files[:0]
		for _, file := range files {
			absolute := filepath.Join(root, filepath.FromSlash(file.RelativePath))
			excluded := false
			for _, localeRoot := range localeRoots {
				if pathContains(localeRoot, absolute) {
					excluded = true
					break
				}
			}
			if !excluded {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}
	if filepath.Clean(root) == filepath.Clean(repositoryRoot) {
		filtered := files[:0]
		for _, file := range files {
			if file.RelativePath != projectChangelogFile {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}
	var openAPIIssues []Issue
	model.openAPIContracts, openAPIIssues = discoverOpenAPIContracts(root, options.Excludes, overlay)
	model.Issues = append(model.Issues, openAPIIssues...)
	for _, file := range files {
		document := createDocument(file, root, staleDays, now, &model.Issues, overlay)
		if document == nil {
			continue
		}
		model.Documents = append(model.Documents, document)
		model.DocByPath[document.SourcePath] = document
	}
	if changelog, issue := loadProjectChangelog(repositoryRoot, staleDays, now, portalUI(model).Text("type.changelog")); issue != nil {
		model.Issues = append(model.Issues, *issue)
	} else {
		model.ProjectChangelog = changelog
	}
	sort.SliceStable(model.Documents, func(i, j int) bool { return documentLess(model.Documents[i], model.Documents[j]) })
	assignUniqueOutputPaths(model)
	usedOutputs := map[string]struct{}{}
	for _, document := range model.Documents {
		usedOutputs[strings.ToLower(document.OutputPath)] = struct{}{}
	}
	for _, candidate := range []string{"health.html", "documentation-health.html", "_project-docs/health.html"} {
		if _, exists := usedOutputs[strings.ToLower(candidate)]; !exists {
			model.HealthOutputPath = candidate
			break
		}
	}
	model.Directories = buildDirectories(model.Documents)
	model.Collections = buildCollections(model.Documents)
	for _, document := range model.Documents {
		validateDocumentBasics(model, document)
	}
	validateGlobalStructure(model)
	validateBuiltinSectionConfiguration(model)
	validateSectionManifests(model)
	resolveLinks(model)
	validateArchitectureDocuments(model)
	connectUseCasesAndModules(model)
	validateMermaidDocuments(model)
	model.Knowledge = buildKnowledgeModel(model)
	validateTypedKnowledge(model)
	buildScreenKnowledge(model)
	model.Risks = buildRisks(model)
	model.RoadmapStages = buildRoadmapStages(model)
	validateRoadmapCompletion(model)
	projectTitle := siteConfig.Title
	if strings.TrimSpace(options.Title) != "" {
		projectTitle = options.Title
	}
	model.Project = buildProjectInfo(model, projectTitle)
	model.CurrentStatus = buildCurrentStatus(model)
	model.Stats = buildStats(model)
	model.SearchIndex = buildSearchIndex(model)
	for index := range model.SearchIndex {
		model.SearchIndex[index].TypeLabel = localizedTypeLabel(model, model.SearchIndex[index].Type)
	}
	return model, nil
}

func validateBuiltinSectionConfiguration(model *Model) {
	config := model.SiteConfig.Project
	if config.Locale == "" {
		model.Issues = append(model.Issues, newIssue("warning", "missing-project-locale", "Set project.locale in .toudocu/config.yml for localized built-in sections.", ".toudocu/config.yml", 0))
	}
	complete := len(config.Sections) == len(BuiltinSections)
	if !complete {
		model.Issues = append(model.Issues, newIssue("warning", "incomplete-project-sections", "Define every project.sections entry in .toudocu/config.yml for localized built-in sections.", ".toudocu/config.yml", 0))
	}
	if config.Locale == "" || !complete {
		return
	}
	for _, spec := range BuiltinSections {
		document := model.DocByPath[path.Join(spec.SourceDir, spec.EntryFile)]
		if document == nil {
			continue
		}
		if canonicalText(document.Title) != canonicalText(config.Sections[spec.Type]) {
			addDocumentIssue(model, document, newIssue("warning", "builtin-section-title-mismatch", fmt.Sprintf("The built-in section H1 must match project.sections.%s.", spec.Type), document.SourcePath, 0))
		}
	}
}

func directoryLabel(model *Model, directory string) string {
	if section := sectionTypeForPath(directory); section != "" {
		if spec, ok := sectionSpec(section); ok {
			return spec.EnglishTitle
		}
	}
	base := path.Base(directory)
	label := strings.ReplaceAll(strings.ReplaceAll(base, "-", " "), "_", " ")
	if label == "" {
		return localizedTypeLabel(model, "document")
	}
	runes := []rune(label)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func directoryHasSourceIndex(model *Model, directory string) bool {
	if directory == "screens" && len(model.Knowledge.Screens) > 0 {
		return true
	}
	if directory == "flows" && len(model.Knowledge.PlayableFlows) > 0 {
		return true
	}
	return model.DocByPath[path.Join(directory, "index.md")] != nil
}
