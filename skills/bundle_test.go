package skills

import (
	"strings"
	"testing"
)

func TestLoadContainsCompleteSkill(t *testing.T) {
	bundle, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if bundle.ID != SkillID || bundle.Version != "0.0.6" || len(bundle.Files) < 20 || len(bundle.Checksum) != 64 {
		t.Fatalf("unexpected bundle: %#v", bundle)
	}
	foundSkill, foundReference, foundArchitectureGate := false, false, false
	foundScreenModel, foundWorkItemModel := false, false
	foundWritingQuality, foundEnglishGuidance := false, false
	foundTriggerEvals, foundMigration := false, false
	foundEnglish, foundRussian := false, false
	var skillText, workflowText, documentModelText, workItemText, writingText, openAIText, triggerCSV string
	var embeddedText strings.Builder
	for _, file := range bundle.Files {
		if strings.Contains(file.Path, "..") || len(file.Data) == 0 {
			t.Fatalf("invalid bundled file %q", file.Path)
		}
		embeddedText.Write(file.Data)
		switch file.Path {
		case "SKILL.md":
			foundSkill = true
			skillText = string(file.Data)
		case "references/workflows.md":
			foundReference = true
			workflowText = string(file.Data)
		case "references/document-model.md":
			documentModelText = string(file.Data)
		case "references/architecture-gate.md":
			foundArchitectureGate = true
		case "references/screen-model.md":
			foundScreenModel = true
		case "references/work-item-model.md":
			foundWorkItemModel = true
			workItemText = string(file.Data)
		case "references/migrations/v1-to-v2.md":
			foundMigration = true
		case "references/writing-quality.md":
			foundWritingQuality = true
			writingText = string(file.Data)
		case "assets/project-guidance/en.md":
			foundEnglishGuidance = true
		case "evals/trigger-prompts.csv":
			foundTriggerEvals = true
			triggerCSV = string(file.Data)
		case "agents/openai.yaml":
			openAIText = string(file.Data)
		case "assets/project-guidance/ru.md":
			t.Fatal("managed agent guidance must remain English-only")
		case "assets/templates/en/index.md":
			foundEnglish = true
		case "assets/templates/ru/index.md":
			foundRussian = true
		}
	}
	if !foundSkill || !foundReference || !foundArchitectureGate || !foundScreenModel || !foundWorkItemModel || !foundMigration || !foundWritingQuality || !foundEnglishGuidance || !foundTriggerEvals || !foundEnglish || !foundRussian {
		t.Fatal("bundle does not include metadata, review references, English guidance, trigger evals, and both template locales")
	}
	if !strings.Contains(skillText, "description: >-") || !strings.Contains(skillText, "references/writing-quality.md") {
		t.Fatal("skill metadata or reader-first routing is missing")
	}
	for _, forbidden := range []string{"Context7", "lumenikoly/toudocu", "unversioned documentation"} {
		if strings.Contains(embeddedText.String(), forbidden) {
			t.Errorf("embedded skill retains external documentation instruction %q", forbidden)
		}
	}
	if !strings.Contains(writingText, "`WRITE001`") || !strings.Contains(writingText, "`WRITE010`") {
		t.Fatal("writing-quality gate is incomplete")
	}
	if !strings.Contains(workflowText, "### Free-form drafts") || !strings.Contains(workflowText, "Do not confuse a document in `drafts/`") || !strings.Contains(documentModelText, "`drafts/**/*.md`") {
		t.Fatal("free-form draft guidance is incomplete")
	}
	if strings.Contains(openAIText, "$toudocu init") || strings.Contains(openAIText, "task verify --run") {
		t.Fatal("default prompt must not infer initialization or executable verification")
	}
	if strings.Count(triggerCSV, ",true,") != 10 || strings.Count(triggerCSV, ",false,") != 10 {
		t.Fatal("trigger evaluation dataset is incomplete")
	}
	normalizedWorkItem := strings.Join(strings.Fields(workItemText), " ")
	for _, expected := range []string{
		"design the complete decomposition and dependency graph",
		"every declared dependency receives an earlier ID than its dependent",
		"ID order never creates a dependency",
		"portal renders a parent task's current subtree recursively",
	} {
		if !strings.Contains(normalizedWorkItem, expected) {
			t.Errorf("work-item guidance does not contain %q", expected)
		}
	}
}

func TestLoadReturnsIndependentData(t *testing.T) {
	first, _ := Load()
	first.Files[0].Data[0] ^= 0xff
	second, err := Load()
	if err != nil || first.Files[0].Data[0] == second.Files[0].Data[0] {
		t.Fatal("bundle data was not copied")
	}
}
