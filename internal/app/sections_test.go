package toudocu

import (
	"strings"
	"testing"
)

func TestBuiltinSectionsStableOrderAndLookups(t *testing.T) {
	want := []SectionType{SectionArchitecture, SectionModules, SectionUseCases, SectionFlows, SectionScreens, SectionDecisions, SectionContracts, SectionQuality, SectionRunbooks, SectionReference, SectionWork, SectionDrafts, SectionGuides}
	if len(BuiltinSections) != len(want) {
		t.Fatalf("sections: %#v", BuiltinSections)
	}
	for index, section := range want {
		if BuiltinSections[index].Type != section {
			t.Fatalf("section %d = %q, want %q", index, BuiltinSections[index].Type, section)
		}
		if got := sectionTypeForPath(BuiltinSections[index].SourceDir + "/entry.md"); got != section {
			t.Fatalf("source lookup = %q, want %q", got, section)
		}
		if got, ok := sectionSpec(section); !ok || got != BuiltinSections[index] {
			t.Fatalf("type lookup = %#v, %v", got, ok)
		}
	}
	if BuiltinSections[3].Route != "processes" || BuiltinSections[3].EnglishTitle != "Processes" {
		t.Fatal("flows section contract changed")
	}
	if BuiltinSections[11].Route != "drafts" || BuiltinSections[11].EnglishTitle != "Drafts" {
		t.Fatal("drafts section contract changed")
	}
}

func TestProjectLocaleConfiguration(t *testing.T) {
	for input, want := range map[string]string{"en-GB": "en-GB", "pt-br": "pt-BR", "sr-latn": "sr-Latn", "de-1901": "de-1901"} {
		config, err := parseSiteConfig([]byte(localeConfigForTest(input)))
		if err != nil || config.Project.DefaultLocale != want {
			t.Fatalf("%s: %#v, %v", input, config.Project, err)
		}
	}
	for _, input := range []string{"???", "Russian language", "ru_", "e", "en-US-extra-"} {
		if _, err := parseSiteConfig([]byte(localeConfigForTest(input))); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}

func localeConfigForTest(locale string) string {
	var config strings.Builder
	config.WriteString("documentationVersion: 3\nproject:\n  defaultLocale: " + locale + "\nlocales:\n  " + locale + ":\n    root: docs\n    sections:\n")
	for _, spec := range BuiltinSections {
		config.WriteString("      " + string(spec.Type) + ": " + spec.EnglishTitle + "\n")
	}
	return config.String()
}

func TestMissingProjectConfigurationUsesEnglishAndWarning(t *testing.T) {
	root, docs := configFixture(t)
	model := buildConfigFixture(t, root, docs, "")
	if modelDirectoryLabel(model, "architecture") != "Architecture" {
		t.Fatal("English fallback is missing")
	}
	page := pageShell(model, "index.html", "x", "", "", "")
	if !strings.Contains(page, `<html lang="en"`) {
		t.Fatal("missing locale must render en")
	}
	if !strings.Contains(renderDashboard(model), `<span class="beta-badge">Beta</span>`) {
		t.Fatal("portal header must identify the beta release stage")
	}
}
