package toudocu_test

import (
	"os"
	"strings"
	"testing"

	"toudocu"
)

func TestPublicAPIFacadeDelegatesToInternalImplementation(t *testing.T) {
	if toudocu.Version != "0.0.6" {
		t.Fatalf("Version = %q, want 0.0.6", toudocu.Version)
	}
	if actual := toudocu.ClassifyDocument("architecture/overview.md"); actual != "architecture" {
		t.Fatalf("ClassifyDocument() = %q, want architecture", actual)
	}
}

func TestRootDocumentationMatchesPublishedState(t *testing.T) {
	english, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	russian, err := os.ReadFile("README.ru.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(english)
	for _, forbidden := range []string{"github.com/your-org/", "scaffold module payments"} {
		if strings.Contains(text, forbidden) || strings.Contains(string(russian), forbidden) {
			t.Errorf("README files contain stale placeholder or invalid example %q", forbidden)
		}
	}
	for _, expected := range []string{
		"## Supported Markdown",
		"scaffold module MOD-PAYMENTS",
		"## Public Go API",
		"[Toudocu source documentation](docs-en/index.md)",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("English README does not contain %q", expected)
		}
	}
	for _, expected := range []string{
		"## Поддерживаемый Markdown",
		"## Публичный Go API",
		"[Исходная документация Toudocu](docs/index.md)",
	} {
		if !strings.Contains(string(russian), expected) {
			t.Errorf("Russian README does not contain %q", expected)
		}
	}
}
