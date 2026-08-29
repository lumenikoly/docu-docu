package toudocu

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultFooterURL = "https://lumenikoly.github.io/toudocu/"

const currentDocumentationVersion = 3

// FooterConfig configures the escaped footer text and its optional HTTPS link.
type FooterConfig struct {
	Text        string
	URL         string
	defaultText bool
}

// HeroConfig configures the dashboard hero.
type HeroConfig struct {
	Enabled bool
	Image   string
}

type ChangesConfig struct {
	DefaultBaseRef       string
	RenameSimilarity     int
	IncludeTaskArtifacts bool
	IncludeAssets        bool
	SemanticDiff         bool
	RenderedDiff         bool
	MaxSourceDiffBytes   int
	MaxRenderedFileBytes int
	Exclude              []string
}

// SiteConfig configures the generated portal's built-in appearance and branding.
type SiteConfig struct {
	DocumentationVersion int
	Title                string
	Logo                 string
	Favicon              string
	Theme                string
	ColorScheme          string
	Accent               string
	Density              string
	ContentWidth         string
	Footer               FooterConfig
	Hero                 HeroConfig
	Changes              ChangesConfig
	Project              ProjectConfig
	// Locales describes peer documentation roots. A Toudocu model remains monolingual.
	Locales      map[string]LocaleProfile
	localeErrors map[string]string
}

// ProjectConfig controls stable built-in section names for one portal locale.
type ProjectConfig struct {
	Locale        string // active locale selected from Locales
	DefaultLocale string
	Sections      map[SectionType]string
}

// LocaleProfile configures one independently checked documentation tree.
// Root is relative to repository root and Sections must name every built-in
// section for Locale.
type LocaleProfile struct {
	Root     string
	Sections map[SectionType]string
}

func defaultSiteConfig() SiteConfig {
	return SiteConfig{
		DocumentationVersion: 1,
		Theme:                "classic",
		ColorScheme:          "system",
		Accent:               "indigo",
		Density:              "comfortable",
		ContentWidth:         "standard",
		Footer: FooterConfig{
			URL:         defaultFooterURL,
			defaultText: true,
		},
		Hero:    HeroConfig{Enabled: true},
		Changes: ChangesConfig{RenameSimilarity: 60, IncludeTaskArtifacts: true, IncludeAssets: true, SemanticDiff: true, RenderedDiff: true, MaxSourceDiffBytes: 2 * 1024 * 1024, MaxRenderedFileBytes: 1024 * 1024},
	}
}

type configScalar struct {
	value  string
	line   int
	quoted bool
}

func parseConfigScalar(raw string, line int) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, nil
	}
	if strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "-") ||
		strings.HasPrefix(raw, "&") || strings.HasPrefix(raw, "*") || raw == "|" || raw == ">" {
		return "", false, fmt.Errorf("config.yml:%d: unsupported YAML construct", line)
	}
	if raw[0] == '"' || raw[0] == '\'' {
		quote := raw[0]
		end := -1
		escaped := false
		for i := 1; i < len(raw); i++ {
			if quote == '"' && raw[i] == '\\' && !escaped {
				escaped = true
				continue
			}
			if raw[i] == quote && !escaped {
				end = i
				break
			}
			escaped = false
		}
		if end < 0 || strings.TrimSpace(raw[end+1:]) != "" && !strings.HasPrefix(strings.TrimSpace(raw[end+1:]), "#") {
			return "", false, fmt.Errorf("config.yml:%d: invalid string", line)
		}
		quoted := raw[:end+1]
		if quote == '\'' {
			return strings.ReplaceAll(quoted[1:len(quoted)-1], "''", "'"), true, nil
		}
		value, err := strconv.Unquote(quoted)
		if err != nil {
			return "", false, fmt.Errorf("config.yml:%d: invalid string: %v", line, err)
		}
		return value, true, nil
	}
	inURL := false
	for i := 0; i < len(raw); i++ {
		if i+2 < len(raw) && raw[i:i+3] == "://" {
			inURL = true
		}
		if raw[i] == '#' && (i == 0 || raw[i-1] == ' ') {
			raw = strings.TrimSpace(raw[:i])
			break
		}
		if raw[i] == ':' && !inURL {
			return "", false, fmt.Errorf("config.yml:%d: a string containing a colon must be quoted", line)
		}
	}
	return strings.TrimSpace(raw), false, nil
}

func parseSiteConfig(data []byte) (SiteConfig, error) {
	config := defaultSiteConfig()
	values := map[string]configScalar{}
	changeExcludes := []string{}
	stack := []string{}
	for index, rawLine := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := index + 1
		if strings.TrimSpace(rawLine) == "" || strings.HasPrefix(strings.TrimSpace(rawLine), "#") {
			continue
		}
		if strings.Contains(rawLine, "\t") {
			return config, fmt.Errorf("config.yml:%d: tabs are not allowed in indentation", line)
		}
		indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))
		if indent%2 != 0 || indent > 6 {
			return config, fmt.Errorf("config.yml:%d: invalid indentation", line)
		}
		level := indent / 2
		if level > len(stack) {
			return config, fmt.Errorf("config.yml:%d: invalid nesting", line)
		}
		text := strings.TrimSpace(rawLine)
		if strings.HasPrefix(text, "- ") && strings.Join(stack, ".") == "changes.exclude" {
			value, _, err := parseConfigScalar(strings.TrimSpace(strings.TrimPrefix(text, "- ")), line)
			if err != nil || value == "" {
				return config, fmt.Errorf("config.yml:%d: invalid changes.exclude", line)
			}
			changeExcludes = append(changeExcludes, value)
			continue
		}
		colon := strings.IndexByte(text, ':')
		if colon <= 0 {
			return config, fmt.Errorf("config.yml:%d: expected a key followed by a colon", line)
		}
		key := strings.TrimSpace(text[:colon])
		if strings.ContainsAny(key, " {}[]&*!|>'\"") {
			return config, fmt.Errorf("config.yml:%d: invalid key %q", line, key)
		}
		stack = stack[:level]
		rawValue := strings.TrimSpace(text[colon+1:])
		if strings.HasPrefix(rawValue, "#") {
			rawValue = ""
		}
		path := strings.Join(append(append([]string{}, stack...), key), ".")
		if _, duplicate := values[path]; duplicate {
			return config, fmt.Errorf("config.yml:%d: duplicate key %q", line, path)
		}
		if rawValue == "" {
			values[path] = configScalar{line: line}
			stack = append(stack, key)
			continue
		}
		value, quoted, err := parseConfigScalar(rawValue, line)
		if err != nil {
			return config, err
		}
		values[path] = configScalar{value: value, line: line, quoted: quoted}
	}

	allowedMaps := map[string]bool{"site": true, "site.footer": true, "site.hero": true, "changes": true, "changes.exclude": true, "project": true, "locales": true}
	allowedScalars := map[string]bool{
		"documentationVersion": true,
		"site.title":           true, "site.logo": true, "site.favicon": true, "site.theme": true,
		"site.colorScheme": true, "site.accent": true, "site.density": true, "site.contentWidth": true,
		"site.footer.text": true, "site.footer.url": true, "site.hero.enabled": true, "site.hero.image": true,
		"changes.defaultBaseRef": true, "changes.renameSimilarity": true, "changes.includeTaskArtifacts": true,
		"changes.includeAssets": true, "changes.semanticDiff": true, "changes.renderedDiff": true,
		"changes.maxSourceDiffBytes": true, "changes.maxRenderedFileBytes": true,
		"project.defaultLocale": true,
	}
	localeKeys := map[string]string{}
	localeErrors := map[string]string{}
	for path, scalar := range values {
		parts := strings.Split(path, ".")
		if len(parts) < 2 || parts[0] != "locales" {
			continue
		}
		locale, ok := normalizeLocale(parts[1])
		if !ok {
			// Keep translation-specific errors deferred until that root is chosen.
			continue
		}
		if previous, duplicate := localeKeys[locale]; duplicate && previous != parts[1] {
			localeErrors[locale] = "duplicate normalized locale " + locale
			continue
		}
		localeKeys[locale] = parts[1]
		allowedMaps["locales."+parts[1]] = true
		allowedMaps["locales."+parts[1]+".sections"] = true
		allowedScalars["locales."+parts[1]+".root"] = true
		for _, spec := range BuiltinSections {
			allowedScalars["locales."+parts[1]+".sections."+string(spec.Type)] = true
		}
		_ = scalar
	}
	if _, hasCustomText := values["site.footer.text"]; hasCustomText {
		config.Footer.defaultText = false
		if _, hasCustomURL := values["site.footer.url"]; !hasCustomURL {
			config.Footer.URL = ""
		}
	}
	for key, scalar := range values {
		if scalar.value == "" && allowedMaps[key] {
			continue
		}
		if !allowedScalars[key] {
			if strings.HasPrefix(key, "locales.") {
				// Translation profiles are selected lazily. This lets a canonical
				// check remain independent from an unfinished locale profile.
				continue
			}
			return config, fmt.Errorf("config.yml:%d: unknown key %q", scalar.line, key)
		}
		isBoolean := key == "site.hero.enabled" || strings.HasPrefix(key, "changes.include") || key == "changes.semanticDiff" || key == "changes.renderedDiff"
		if !isBoolean && !scalar.quoted && (scalar.value == "true" || scalar.value == "false") {
			return config, fmt.Errorf("config.yml:%d: %s must be a string", scalar.line, key)
		}
		switch key {
		case "documentationVersion":
			value, err := strconv.Atoi(scalar.value)
			if err != nil || value < 1 {
				return config, fmt.Errorf("config.yml:%d: documentationVersion must be a positive integer", scalar.line)
			}
			config.DocumentationVersion = value
		case "project.defaultLocale":
			locale, ok := normalizeLocale(scalar.value)
			if !ok {
				return config, fmt.Errorf("config.yml:%d: project.defaultLocale must be a valid BCP-47-style locale", scalar.line)
			}
			config.Project.DefaultLocale = locale
		case "locales":
		case "site.title":
			config.Title = scalar.value
		case "site.logo":
			config.Logo = scalar.value
		case "site.favicon":
			config.Favicon = scalar.value
		case "site.theme":
			config.Theme = scalar.value
		case "site.colorScheme":
			config.ColorScheme = scalar.value
		case "site.accent":
			config.Accent = scalar.value
		case "site.density":
			config.Density = scalar.value
		case "site.contentWidth":
			config.ContentWidth = scalar.value
		case "site.footer.text":
			config.Footer.Text = scalar.value
		case "site.footer.url":
			config.Footer.URL = scalar.value
		case "site.hero.image":
			config.Hero.Image = scalar.value
		case "site.hero.enabled":
			if scalar.quoted || scalar.value != "true" && scalar.value != "false" {
				return config, fmt.Errorf("config.yml:%d: site.hero.enabled must be a boolean", scalar.line)
			}
			config.Hero.Enabled = scalar.value == "true"
		case "changes.defaultBaseRef":
			config.Changes.DefaultBaseRef = scalar.value
		case "changes.renameSimilarity":
			value, err := strconv.Atoi(scalar.value)
			if err != nil || value < 1 || value > 100 {
				return config, fmt.Errorf("config.yml:%d: changes.renameSimilarity must be between 1 and 100", scalar.line)
			}
			config.Changes.RenameSimilarity = value
		case "changes.maxSourceDiffBytes":
			value, err := strconv.Atoi(scalar.value)
			if err != nil || value < 1 {
				return config, fmt.Errorf("config.yml:%d: changes.maxSourceDiffBytes must be positive", scalar.line)
			}
			config.Changes.MaxSourceDiffBytes = value
		case "changes.maxRenderedFileBytes":
			value, err := strconv.Atoi(scalar.value)
			if err != nil || value < 1 {
				return config, fmt.Errorf("config.yml:%d: changes.maxRenderedFileBytes must be positive", scalar.line)
			}
			config.Changes.MaxRenderedFileBytes = value
		case "changes.includeTaskArtifacts", "changes.includeAssets", "changes.semanticDiff", "changes.renderedDiff":
			if scalar.quoted || scalar.value != "true" && scalar.value != "false" {
				return config, fmt.Errorf("config.yml:%d: %s must be a boolean", scalar.line, key)
			}
			value := scalar.value == "true"
			switch key {
			case "changes.includeTaskArtifacts":
				config.Changes.IncludeTaskArtifacts = value
			case "changes.includeAssets":
				config.Changes.IncludeAssets = value
			case "changes.semanticDiff":
				config.Changes.SemanticDiff = value
			case "changes.renderedDiff":
				config.Changes.RenderedDiff = value
			}
		}
	}
	if len(localeKeys) > 0 {
		config.Locales = map[string]LocaleProfile{}
		config.localeErrors = localeErrors
		for locale, rawLocale := range localeKeys {
			profile := LocaleProfile{Sections: map[SectionType]string{}}
			if root, ok := values["locales."+rawLocale+".root"]; ok {
				profile.Root = root.value
			}
			for _, spec := range BuiltinSections {
				if title, ok := values["locales."+rawLocale+".sections."+string(spec.Type)]; ok {
					profile.Sections[spec.Type] = title.value
				}
			}
			config.Locales[locale] = profile
		}
	}
	if config.DocumentationVersion == currentDocumentationVersion {
		for path, scalar := range values {
			parts := strings.Split(path, ".")
			if len(parts) >= 2 && parts[0] == "locales" {
				if _, ok := normalizeLocale(parts[1]); !ok {
					return config, fmt.Errorf("config.yml:%d: locale key %q must be a valid BCP-47-style locale", scalar.line, parts[1])
				}
			}
		}
		for locale, message := range localeErrors {
			if message != "" {
				return config, fmt.Errorf("LOCALE_INVALID: locales.%s: %s", locale, message)
			}
		}
	}
	for locale, profile := range config.Locales {
		completeLegacyDrafts(locale, profile.Sections)
		config.Locales[locale] = profile
	}
	config.Changes.Exclude = changeExcludes
	if _, ok := values["site"]; !ok {
		if _, changesOnly := values["changes"]; changesOnly || values["documentationVersion"].line > 0 || values["project"].line > 0 || values["locales"].line > 0 {
			return config, validateSiteConfig(config)
		}
		return config, fmt.Errorf("config.yml: root site map is missing")
	}
	return config, validateSiteConfig(config)
}

func documentationVersionIssue(config SiteConfig) *Issue {
	version := config.DocumentationVersion
	if version < currentDocumentationVersion {
		migration := "v1-to-v2"
		if version == 2 {
			migration = "v2-to-v3"
		}
		return &Issue{
			Severity: "error", Code: "DOCS_MIGRATION_REQUIRED",
			Message:   fmt.Sprintf("Documentation version %d must be migrated to version %d.", version, currentDocumentationVersion),
			Migration: migration, DocumentPath: ".toudocu/config.yml",
		}
	}
	if version > currentDocumentationVersion {
		return &Issue{
			Severity: "error", Code: "DOCUMENTATION_VERSION_UNSUPPORTED",
			Message:      fmt.Sprintf("Documentation version %d is newer than supported version %d; update Toudocu.", version, currentDocumentationVersion),
			DocumentPath: ".toudocu/config.yml",
		}
	}
	return nil
}

func documentationVersionError(issue *Issue) error {
	if issue == nil {
		return nil
	}
	if issue.Migration != "" {
		return fmt.Errorf("%s: Migration: %s", issue.Code, issue.Migration)
	}
	return fmt.Errorf("%s: %s", issue.Code, issue.Message)
}

func requireCurrentDocumentationVersion(options Options) error {
	root, err := filepath.Abs(options.InputDirectory)
	if err != nil {
		return err
	}
	repositoryRoot := options.RepositoryRoot
	if repositoryRoot == "" {
		repositoryRoot = filepath.Dir(root)
	}
	config, _, err := loadSiteConfig(repositoryRoot)
	if err != nil {
		return err
	}
	return documentationVersionError(documentationVersionIssue(config))
}

func completeLegacyDrafts(locale string, sections map[SectionType]string) {
	if sections == nil {
		return
	}
	if _, exists := sections[SectionDrafts]; exists {
		return
	}
	for _, spec := range BuiltinSections {
		if spec.Type != SectionDrafts && strings.TrimSpace(sections[spec.Type]) == "" {
			return
		}
	}
	sections[SectionDrafts] = defaultSectionTitle(locale, SectionDrafts)
}

func enumValue(field, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("config.yml: invalid site.%s value %q (allowed: %s)", field, value, strings.Join(allowed, ", "))
}

func validateSiteConfig(config SiteConfig) error {
	if config.DocumentationVersion == currentDocumentationVersion {
		if config.Project.DefaultLocale == "" {
			return fmt.Errorf("config.yml: project.defaultLocale is required")
		}
		if _, ok := config.Locales[config.Project.DefaultLocale]; !ok {
			return fmt.Errorf("config.yml: project.defaultLocale must reference a configured locale")
		}
	}
	if err := enumValue("theme", config.Theme, "classic", "paper", "terminal"); err != nil {
		return err
	}
	if err := enumValue("colorScheme", config.ColorScheme, "light", "dark", "system"); err != nil {
		return err
	}
	if err := enumValue("accent", config.Accent, "indigo", "blue", "teal", "green", "amber", "rose", "violet"); err != nil {
		return err
	}
	if err := enumValue("density", config.Density, "compact", "comfortable"); err != nil {
		return err
	}
	if err := enumValue("contentWidth", config.ContentWidth, "narrow", "standard", "wide"); err != nil {
		return err
	}
	if config.Footer.URL != "" {
		parsed, err := url.ParseRequestURI(config.Footer.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || strings.ContainsAny(config.Footer.URL, "\r\n\t <>\"'") {
			return fmt.Errorf("config.yml: site.footer.url must be a safe HTTPS URL")
		}
	}
	return nil
}

func validateBrandAsset(repositoryRoot, configuredPath, kind string) (string, string, error) {
	if configuredPath == "" {
		return "", "", nil
	}
	if filepath.IsAbs(configuredPath) || strings.Contains(configuredPath, "\\") {
		return "", "", fmt.Errorf("config.yml: site.%s must be a relative path inside assets/", kind)
	}
	clean := filepath.Clean(filepath.FromSlash(configuredPath))
	if clean == "." || clean == "assets" || !strings.HasPrefix(clean, "assets"+string(filepath.Separator)) {
		return "", "", fmt.Errorf("config.yml: site.%s must be inside .toudocu/assets/", kind)
	}
	relative := strings.TrimPrefix(clean, "assets"+string(filepath.Separator))
	if relative == "" || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("config.yml: site.%s escapes .toudocu/assets/", kind)
	}
	assetsRoot := filepath.Join(repositoryRoot, ".toudocu", "assets")
	for _, directory := range []string{filepath.Join(repositoryRoot, ".toudocu"), assetsRoot} {
		info, err := os.Lstat(directory)
		if err != nil {
			if os.IsNotExist(err) {
				return "", "", fmt.Errorf("config.yml: site.%s asset not found: %s", kind, configuredPath)
			}
			return "", "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", "", fmt.Errorf("config.yml: symbolic links are not allowed in site.%s path: %s", kind, configuredPath)
		}
	}
	current := assetsRoot
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return "", "", fmt.Errorf("config.yml: site.%s asset not found: %s", kind, configuredPath)
			}
			return "", "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("config.yml: symbolic links are not allowed for site.%s: %s", kind, configuredPath)
		}
	}
	info, err := os.Stat(current)
	if err != nil || !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("config.yml: site.%s must point to a regular file", kind)
	}
	extension := strings.ToLower(filepath.Ext(current))
	if extension == "" {
		extension = ".bin"
	}
	return current, "assets/branding/" + kind + extension, nil
}

func loadSiteConfig(repositoryRoot string) (SiteConfig, map[string]string, error) {
	configPath := filepath.Join(repositoryRoot, ".toudocu", "config.yml")
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return defaultSiteConfig(), map[string]string{}, nil
	}
	if err != nil {
		return SiteConfig{}, nil, fmt.Errorf("could not read .toudocu/config.yml: %w", err)
	}
	config, err := parseSiteConfig(data)
	if err != nil {
		return SiteConfig{}, nil, err
	}
	if config.DocumentationVersion == currentDocumentationVersion {
		roots := map[string]string{}
		for locale, profile := range config.Locales {
			root, rootErr := safeTranslationRoot(repositoryRoot, profile.Root)
			if rootErr != nil {
				return SiteConfig{}, nil, fmt.Errorf("LOCALE_PROFILE_INVALID: locales.%s.root: %w", locale, rootErr)
			}
			for otherRoot, otherLocale := range roots {
				if pathContains(root, otherRoot) || pathContains(otherRoot, root) {
					return SiteConfig{}, nil, fmt.Errorf("LOCALE_ROOT_COLLISION: locales.%s overlaps locales.%s", locale, otherLocale)
				}
			}
			roots[root] = locale
			if len(profile.Sections) != len(BuiltinSections) {
				return SiteConfig{}, nil, fmt.Errorf("LOCALE_PROFILE_INCOMPLETE: locales.%s.sections must contain every built-in section", locale)
			}
		}
	}
	branding := map[string]string{}
	for kind, configured := range map[string]string{"logo": config.Logo, "favicon": config.Favicon, "hero": config.Hero.Image} {
		source, output, assetErr := validateBrandAsset(repositoryRoot, configured, kind)
		if assetErr != nil {
			return SiteConfig{}, nil, assetErr
		}
		if source != "" {
			branding[output] = source
		}
	}
	return config, branding, nil
}

// selectLocaleProfile selects exactly one configured locale root. The
// model remains monolingual; all configured roots are peers.
func selectLocaleProfile(config *SiteConfig, repositoryRoot, inputRoot string) ([]string, error) {
	inputRoot = filepath.Clean(inputRoot)
	validRoots := []string{}
	selected := ""
	for locale, profile := range config.Locales {
		root, err := safeTranslationRoot(repositoryRoot, profile.Root)
		if err != nil {
			continue
		}
		validRoots = append(validRoots, root)
		if filepath.Clean(root) == inputRoot {
			selected = locale
		}
	}
	if selected == "" {
		if config.DocumentationVersion == currentDocumentationVersion {
			return nil, fmt.Errorf("LOCALE_ROOT_NOT_CONFIGURED: input root must match locales.<locale>.root")
		}
		return validRoots, nil
	}
	profile := config.Locales[selected]
	if config.localeErrors[selected] != "" {
		return nil, fmt.Errorf("%s", config.localeErrors[selected])
	}
	root, err := safeTranslationRoot(repositoryRoot, profile.Root)
	if err != nil {
		return nil, fmt.Errorf("LOCALE_PROFILE_INVALID: %w", err)
	}
	if len(profile.Sections) != len(BuiltinSections) {
		return nil, fmt.Errorf("LOCALE_PROFILE_INCOMPLETE: locales.%s.sections must contain every built-in section", selected)
	}
	for _, spec := range BuiltinSections {
		if strings.TrimSpace(profile.Sections[spec.Type]) == "" {
			return nil, fmt.Errorf("LOCALE_PROFILE_INCOMPLETE: locales.%s.sections.%s is empty", selected, spec.Type)
		}
	}
	for otherLocale, other := range config.Locales {
		if otherLocale == selected {
			continue
		}
		otherRoot, otherErr := safeTranslationRoot(repositoryRoot, other.Root)
		if otherErr != nil {
			continue
		}
		if filepath.Clean(otherRoot) == root || pathContains(root, otherRoot) || pathContains(otherRoot, root) {
			return nil, fmt.Errorf("LOCALE_ROOT_COLLISION: locales.%s overlaps locales.%s", selected, otherLocale)
		}
	}
	config.Project.Locale = selected
	config.Project.Sections = profile.Sections
	return validRoots, nil
}

func safeTranslationRoot(repositoryRoot, configured string) (string, error) {
	if strings.TrimSpace(configured) == "" || filepath.IsAbs(configured) || strings.Contains(configured, "\\") {
		return "", fmt.Errorf("translation root must be a non-empty relative path inside the repository root")
	}
	clean := filepath.Clean(filepath.FromSlash(configured))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("translation root escapes the repository root")
	}
	root := filepath.Join(repositoryRoot, clean)
	if !pathContains(repositoryRoot, root) || filepath.Clean(root) == filepath.Clean(repositoryRoot) {
		return "", fmt.Errorf("translation root must be nested inside the repository root")
	}
	current := repositoryRoot
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symbolic links are not allowed in the translation root: %s", configured)
		}
	}
	return root, nil
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
