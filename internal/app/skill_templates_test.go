package toudocu

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToudocuFlowAndScreenTemplates(t *testing.T) {
	for _, language := range []string{"ru", "en"} {
		t.Run(language, func(t *testing.T) {
			root := t.TempDir()
			docs := filepath.Join(root, "docs")
			writeTestFile(t, docs, "index.md", "# Template fixture\n\nToudocu skill template fixture.\n")
			writeArchitectureOverview(t, docs, "")
			writeTestFile(t, docs, "modules/core.md", `# Core

- Идентификатор: MOD-CORE
- Статус: Запланировано

Core module.

## Бизнес-правила

### BR-CORE-001: Continue

The user can continue from the start screen.
`)

			replacements := map[string]string{
				"{{TRANSITION_ROWS}}":               "| TR-CORE-001 | UC-CORE-01 | Open workspace | Always | SC-CORE-WORKSPACE | DEFAULT | — | — | — | navigation |",
				"{{MODULE_ID}}":                     "MOD-CORE",
				"{{USE_CASE_ID}}":                   "UC-CORE-01",
				"{{USE_CASE_TITLE}}":                "Continue",
				"{{STATUS}}":                        "planned",
				"{{ACTOR}}":                         "User",
				"{{OPTIONAL_SCREENS_METADATA}}":     "screens: SC-CORE-HOME, SC-CORE-WORKSPACE",
				"{{START_SCREEN_ID}}":               "SC-CORE-HOME",
				"{{TERMINAL_SCREEN_IDS}}":           "SC-CORE-WORKSPACE",
				"{{OPTIONAL_ALLOW_CYCLE_METADATA}}": "",
				"{{PRIORITY}}":                      "medium",
				"{{DATE}}":                          "2026-07-28",
				"{{USE_CASE_SUMMARY}}":              "The user opens the workspace from the home page.",
				"{{INPUT}}":                         "A valid request.",
				"{{PRECONDITION}}":                  "The home page is open.",
				"{{MAIN_STEP}}":                     "The user opens the workspace.",
				"{{ERROR_SCENARIOS}}":               "No error scenario is defined.",
				"{{POSTCONDITIONS}}":                "The workspace is open.",
				"{{BUSINESS_RULE_ID}}":              "BR-CORE-001",
				"{{BUSINESS_RULE_REFERENCE}}":       "Continue is allowed.",
				"{{MODULE_TITLE}}":                  "Core",
				"{{MODULE_FILE}}":                   "core.md",
				"{{FLOW_ID}}":                       "FLOW-CORE-01",
				"{{FLOW_TITLE}}":                    "Continue",
				"{{FLOW_SUMMARY}}":                  "Detailed workspace navigation.",
				"{{FLOW_DIAGRAM}}":                  "flowchart TD\n    Home[\"Home\"] -->|Open workspace| Workspace[\"Workspace\"]",
				"{{OPTIONAL_USE_CASES_METADATA}}":   "useCase: UC-CORE-01, UC-CORE-02",
				"{{RELATED_DOCUMENT_LINKS}}":        "- [UC-CORE-01](../use-cases/core.md)\n- [UC-CORE-02](../use-cases/secondary.md)",
				"{{USE_CASE_LINK}}":                 "../use-cases/core.md",
				"{{SCREEN_ID}}":                     "SC-CORE-HOME",
				"{{SCREEN_TITLE}}":                  "Home",
				"{{SCREEN_STATUS}}":                 "planned",
				"{{SCREEN_TYPE}}":                   "page",
				"{{OPTIONAL_ROUTE_METADATA}}":       "route: /",
				"{{OPTIONAL_PREVIEW_METADATA}}":     "",
				"{{OPTIONAL_PARENT_METADATA}}":      "",
				"{{OPTIONAL_COMPONENT_METADATA}}":   "component: web/home/",
				"{{YYYY-MM-DD}}":                    "2026-07-28",
				"{{SCREEN_PURPOSE}}":                "Provides the product entry point.",
				"{{STATE_ROWS}}":                    "| DEFAULT | Default | — |",
				"{{TASK_ID}}":                       "TASK-CORE-001",
				"{{TASK_TITLE}}":                    "Implement continue",
				"{{OPTIONAL_FLOW_METADATA}}":        "flow: FLOW-CORE-01",
				"{{OPTIONAL_TRANSITIONS_METADATA}}": "transitions: TR-CORE-001",
				"{{TRANSITION_ID}}":                 "TR-CORE-001",
				"{{VERIFICATION_REFERENCE}}":        "TestOpenWorkspace",
				"{{RESULT}}":                        "The continue path works.",
				"{{BEFORE}}":                        "The continue path is unavailable.",
				"{{AFTER}}":                         "The continue path opens the workspace.",
				"{{SCOPE_PATH}}":                    "docs",
				"{{OUT_OF_SCOPE}}":                  "Other scenarios.",
				"{{ACCEPTANCE_CRITERION}}":          "Open workspace navigates to SC-CORE-WORKSPACE.",
				"{{PLAN_STEP}}":                     "Implement the documented transition.",
				"{{ACCEPTANCE_COMMAND}}":            "go test ./...",
				"{{ALL_COMMAND}}":                   "go test ./...",
				"{{DOCS_COMMAND}}":                  "go run ./cmd/toudocu check ./docs",
				"{{DOCUMENTATION_IMPACT}}":          "Update the screen map.",
			}
			writeSkillTemplate(t, docs, language, "use-case.md", "use-cases/core.md", replacements)
			writeTestFile(t, docs, "use-cases/secondary.md", `# UC-CORE-02: Review

- Identifier: UC-CORE-02
- Status: Planned
- Module: MOD-CORE

The user reviews the workspace.
`)
			writeSkillTemplate(t, docs, language, "flow.md", "flows/core.md", replacements)
			writeSkillTemplate(t, docs, language, "screen.md", "screens/SC-CORE-HOME.md", replacements)
			writeSkillTemplate(t, docs, language, "work-ready-feature.md", "work/TASK-CORE-001.md", replacements)
			writeTestFile(t, docs, "screens/SC-CORE-WORKSPACE.md", `# SC-CORE-WORKSPACE: Workspace

- Идентификатор: SC-CORE-WORKSPACE
- Тип: Страница
- Модуль: MOD-CORE
- Статус: Запланировано
- Маршрут: /workspace

Terminal screen.
`)

			model, err := BuildDocumentationModel(Options{
				InputDirectory: docs,
				RepositoryRoot: root,
				StaleDays:      0,
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, issue := range model.Issues {
				if issue.Severity == "error" {
					t.Fatalf("instantiated %s skill templates produced an error: %#v", language, issue)
				}
			}
			if len(model.Knowledge.Screens) != 2 || len(model.Knowledge.Transitions) != 1 {
				t.Fatalf("unexpected screen graph: %#v", model.Knowledge)
			}
			if len(model.Knowledge.Flows) != 1 || strings.Join(model.Knowledge.Flows[0].UseCaseIDs, ",") != "UC-CORE-01,UC-CORE-02" {
				t.Fatalf("flow does not contain both use cases: %#v", model.Knowledge.Flows)
			}
			for _, useCase := range model.Knowledge.UseCases {
				if strings.Join(useCase.FlowIDs, ",") != "FLOW-CORE-01" {
					t.Fatalf("%s does not contain the reverse flow relationship: %#v", useCase.ID, useCase.FlowIDs)
				}
			}
			context, err := BuildTaskContext(model, "TASK-CORE-001")
			if err != nil {
				t.Fatal(err)
			}
			if context.Task.FlowID != "FLOW-CORE-01" || len(context.Screens) != 2 || len(context.ScreenTransitions) != 1 {
				t.Fatalf("flow or screens missing from task context: %#v", context)
			}
		})
	}
}

func TestToudocuTemplatesDoNotInventSemanticStructure(t *testing.T) {
	for _, language := range []string{"ru", "en"} {
		screen := readSkillTemplate(t, language, "screen.md")
		for _, placeholder := range []string{"{{STATE_ROWS}}", "{{TRANSITION_ROWS}}"} {
			if !strings.Contains(screen, placeholder) {
				t.Errorf("%s/screen.md does not contain %s", language, placeholder)
			}
		}

		flow := readSkillTemplate(t, language, "flow.md")
		for _, placeholder := range []string{"{{FLOW_DIAGRAM}}", "{{OPTIONAL_USE_CASES_METADATA}}", "{{RELATED_DOCUMENT_LINKS}}"} {
			if !strings.Contains(flow, placeholder) {
				t.Errorf("%s/flow.md does not contain %s", language, placeholder)
			}
		}
		for _, forbidden := range []string{"{{START}}", "{{FINISH}}", "Start --> Finish"} {
			if strings.Contains(flow, forbidden) {
				t.Errorf("%s/flow.md contains default topology %s", language, forbidden)
			}
		}

		decision := readSkillTemplate(t, language, "decision.md")
		if !strings.Contains(decision, "{{CONSEQUENCES}}") {
			t.Errorf("%s/decision.md does not contain CONSEQUENCES", language)
		}
		for _, forbidden := range []string{"{{POSITIVE_CONSEQUENCE}}", "{{NEGATIVE_CONSEQUENCE}}"} {
			if strings.Contains(decision, forbidden) {
				t.Errorf("%s/decision.md requires invented polarity %s", language, forbidden)
			}
		}
	}
}

func TestToudocuWorkItemReferenceAllowsCriteriaAndPlanChecklists(t *testing.T) {
	content, err := os.ReadFile(repositoryPath("skills", "toudocu", "references", "work-item-model.md"))
	if err != nil {
		t.Fatal(err)
	}
	reference := string(content)
	if !strings.Contains(reference, "Checkboxes are allowed in both acceptance criteria and plan.") {
		t.Fatal("work-item-model reference does not allow checkboxes in both acceptance criteria and plan")
	}
	if strings.Contains(reference, "Put checkboxes only in acceptance criteria.") {
		t.Fatal("work-item-model reference still restricts checkboxes to acceptance criteria")
	}
}

func TestToudocuOptionalRelationshipPlaceholders(t *testing.T) {
	for _, language := range []string{"ru", "en"} {
		for _, templateName := range []string{"work-ready-feature.md", "work-ready-technical.md", "work-draft.md"} {
			content := readSkillTemplate(t, language, templateName)
			for _, placeholder := range []string{"{{OPTIONAL_FLOW_METADATA}}", "{{OPTIONAL_SCREENS_METADATA}}"} {
				if !strings.Contains(content, placeholder) {
					t.Errorf("%s/%s does not contain %s", language, templateName, placeholder)
				}
			}
		}
		bug := readSkillTemplate(t, language, "work-ready-bug.md")
		for _, placeholder := range []string{"{{BUG_ID}}", "{{REPRODUCIBILITY}}", "{{REGRESSION_CRITERION}}", "{{OPTIONAL_SCREENS_METADATA}}"} {
			if !strings.Contains(bug, placeholder) {
				t.Errorf("%s/work-ready-bug.md does not contain %s", language, placeholder)
			}
		}
		if content := readSkillTemplate(t, language, "use-case.md"); !strings.Contains(content, "{{OPTIONAL_SCREENS_METADATA}}") {
			t.Errorf("%s/use-case.md does not support screen relationships", language)
		}
		screen := readSkillTemplate(t, language, "screen.md")
		for _, placeholder := range []string{"{{OPTIONAL_ROUTE_METADATA}}", "{{OPTIONAL_COMPONENT_METADATA}}"} {
			if !strings.Contains(screen, placeholder) {
				t.Errorf("%s/screen.md does not contain %s", language, placeholder)
			}
		}
	}
}

func TestToudocuInitContract(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	for _, expected := range []string{
		"`$toudocu init`",
		"[references/init.md](references/init.md)",
		"Never infer `$toudocu init`",
		"agent workflows, not Toudocu Go CLI commands",
	} {
		if !containsNormalized(skill, expected) {
			t.Errorf("SKILL.md does not define explicit init contract %q", expected)
		}
	}

	initReference := readToudocuFile(t, filepath.Join("references", "init.md"))
	for _, expected := range []string{
		"Do not infer initialization",
		"Preflight before changing files",
		"<!-- toudocu:project-guidance:start -->",
		"<!-- toudocu:project-guidance:end -->",
		"duplicated",
		"reversed",
		"conflicting",
		"no `index.md`",
		"`missing-architecture-overview` may be the only error",
		"`<docs-root>/architecture/overview.md`",
		"`missing-project-locale`",
		"Use `assets/project-guidance/en.md` for every project locale",
		"Managed agent instructions\n   remain English",
		"Set its H1 exactly to the resolved `project.sections.architecture`",
		"entry document that existed before init",
		"without removing\n   existing `site`, `changes`, or `locales` settings",
		"as legacy\n   architecture",
		"stop without migrating or rewriting",
		"ordinary project-wide Toudocu check",
		"Do not create a `TASK-*` merely because init is running",
	} {
		if !containsNormalized(initReference, expected) {
			t.Errorf("init reference does not contain %q", expected)
		}
	}
}

func TestToudocuRefreshContract(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	for _, expected := range []string{
		"`$toudocu refresh`",
		"`$toudocu refresh diff`",
		"[references/refresh.md](references/refresh.md)",
		"agent workflows, not Toudocu Go CLI commands",
	} {
		if !containsNormalized(skill, expected) {
			t.Errorf("SKILL.md does not define refresh contract %q", expected)
		}
	}

	refresh := readToudocuFile(t, filepath.Join("references", "refresh.md"))
	for _, expected := range []string{
		"`$toudocu refresh`",
		"`$toudocu refresh diff`",
		"inventory every source Markdown document",
		"`git diff --name-only HEAD --`",
		"`git ls-files --others --exclude-standard`",
		"staged and unstaged tracked changes",
		"Do not compare with a merge-base or\n   default branch",
		"If Git or `HEAD` is unavailable",
		"links, backlinks, stable IDs",
		"Exclude generated portals",
		"does not\n   change code to make a document true",
		"unresolved\n   findings",
		"Creating, deleting, renaming, or merging a document",
		"Update every\n   affected link, ID reference",
		"only when content or declared relationships actually change",
		"Never advance the runbook verification date",
		"Never edit generated portal output as documentation",
		"Obtain independent semantic review",
		"Rebuild a portal only when it is tracked",
		"Refresh never performs initialization",
	} {
		if !containsNormalized(refresh, expected) {
			t.Errorf("refresh reference does not contain %q", expected)
		}
	}

	initReference := readToudocuFile(t, filepath.Join("references", "init.md"))
	if strings.Contains(initReference, "$toudocu refresh") {
		t.Fatal("init reference must not route refresh")
	}

	for _, language := range []string{"en"} {
		guidance := readToudocuFile(t, filepath.Join("assets", "project-guidance", language+".md"))
		for _, expected := range []string{"$toudocu refresh", "$toudocu refresh diff", "HEAD"} {
			if !strings.Contains(guidance, expected) {
				t.Errorf("%s guidance does not contain %q", language, expected)
			}
		}
	}
}

func TestToudocuAgentFeedbackContract(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	for _, expected := range []string{
		"$toudocu feedback",
		"Toudocu discussions",
		"local Agent Feedback queue",
		"[references/agent-feedback.md](references/agent-feedback.md)",
		"Process Agent Feedback only through `toudocu agent next|respond`",
		"operation reference owns validation and delivery",
		"do not run ordinary checks, tests, or builds for feedback",
	} {
		if !containsNormalized(skill, expected) {
			t.Errorf("SKILL.md does not define agent feedback contract %q", expected)
		}
	}
	feedback := readToudocuFile(t, filepath.Join("references", "agent-feedback.md"))
	for _, expected := range []string{
		"toudocu agent next",
		"The response from `agent next` is the sole source of work",
		"does not authorize searching documentation, status, the roadmap, `UC-*`, `TASK-*`, acceptance criteria",
		"`pending=false`, stop successfully without reading or changing files",
		"no task, criterion, or other work outside its messages",
		"until the queue is empty",
		"always complete the current one with `toudocu agent respond`",
		"An open discussion is not pending work",
		"`question`: answer from current evidence and do not change documentation",
		"change `target.path`",
		"unambiguously named in that message",
		"Do not infer a related path",
		"A `document` target permits canonical documentation paths",
		"a `file` target permits safe repository paths",
		"`needs_clarification`",
		"Do not run `check`, tests, lint, build or any other validation command",
		"For `changed`, the message must explicitly say that the requested edits were made",
		"list every path from `changedPaths`",
		"toudocu agent respond",
		"AGENT_RESPONSE_CONFLICT",
		"Never resolve the discussion for the user",
	} {
		if !containsNormalized(feedback, expected) {
			t.Errorf("agent feedback reference does not contain %q", expected)
		}
	}
}

func TestToudocuCompactOperationRouter(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	for _, expected := range []string{
		"| Request | Read | Rule |",
		"[references/init.md](references/init.md)",
		"[references/refresh.md](references/refresh.md)",
		"[references/translate.md](references/translate.md)",
		"[references/agent-feedback.md](references/agent-feedback.md)",
		"$toudocu translate <target-locale> --from <source-locale>",
		"[references/workflows.md](references/workflows.md)",
		"[references/semantic-gate.md](references/semantic-gate.md)",
		"[references/architecture-gate.md](references/architecture-gate.md)",
		"[references/screen-model.md](references/screen-model.md)",
		"[references/work-item-model.md](references/work-item-model.md)",
		"Explicit `$toudocu` operations take precedence",
		"do not add `workflows.md`",
		"Read-only review, analysis, or explanation",
		"Skip `workflows.md` unless CLI or diagnostics are required",
	} {
		if !containsNormalized(skill, expected) {
			t.Errorf("compact router missing %q", expected)
		}
	}
}

func TestToudocuGlobalInvariants(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	for _, expected := range []string{
		"Give repository evidence priority over assumptions",
		"generated portals, builds, reports, and example output as derived artifacts",
		"Never infer `$toudocu init` from missing files, first use",
		"agent workflows, not Toudocu Go CLI commands",
		"Run `task verify --run` only when the user explicitly requests execution",
		"Each configured locale root is an independent documentation and backlog source",
		"Process Agent Feedback only through `toudocu agent next|respond`",
		"Create a durable work item only when the user or repository explicitly requires one",
	} {
		if !containsNormalized(skill, expected) {
			t.Errorf("SKILL.md misses global invariant %q", expected)
		}
	}
}

func TestToudocuArchitectureTemplates(t *testing.T) {
	for _, language := range []string{"en"} {
		t.Run(language, func(t *testing.T) {
			overview := readSkillTemplate(t, language, "architecture-overview.md")
			detail := readSkillTemplate(t, language, "architecture-detail.md")
			for _, expected := range []string{"{{SYSTEM_BOUNDARY}}", "{{ARCHITECTURE_QUESTION_LINKS}}", "{{OPTIONAL_CONTEXT_DIAGRAM}}"} {
				if !strings.Contains(overview, expected) {
					t.Errorf("%s architecture overview does not contain %q", language, expected)
				}
			}
			for _, expected := range []string{"architectureQuestion: {{ARCHITECTURE_QUESTION}}", "{{SHORT_ANSWER}}", "{{SCOPE}}", "{{ADAPTABLE_SECTIONS}}"} {
				if !strings.Contains(detail, expected) {
					t.Errorf("%s architecture detail does not contain %q", language, expected)
				}
			}
			if _, err := os.Stat(repositoryPath("skills", "toudocu", "assets", "templates", language, "architecture.md")); !os.IsNotExist(err) {
				t.Errorf("%s monolithic architecture template still exists: %v", language, err)
			}

			root := t.TempDir()
			docs := filepath.Join(root, "docs")
			writeTestFile(t, docs, "index.md", "# Project\n\nProject documentation.\n")
			replacements := map[string]string{
				"{{PROJECT_TITLE}}":               "Project",
				"{{SYSTEM_BOUNDARY_SUMMARY}}":     "The system serves one documented client.",
				"{{SYSTEM_BOUNDARY}}":             "The system accepts requests from the client and returns results.",
				"{{ARCHITECTURE_QUESTION_LINKS}}": "- [How is runtime responsibility split](runtime.md)",
				"{{OPTIONAL_CONTEXT_DIAGRAM}}":    "",
			}
			writeSkillTemplate(t, docs, language, "architecture-overview.md", "architecture/overview.md", replacements)
			replacements = map[string]string{
				"{{ARCHITECTURE_TITLE}}":    "Runtime responsibility",
				"{{ARCHITECTURE_QUESTION}}": "How is runtime responsibility split",
				"{{SHORT_ANSWER}}":          "One process owns parsing and validation.",
				"{{SCOPE}}":                 "Runtime components inside the process.",
				"{{ADAPTABLE_SECTIONS}}":    "## Components\n\nThe parser produces the validated model.",
			}
			writeSkillTemplate(t, docs, language, "architecture-detail.md", "architecture/runtime.md", replacements)
			model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
			if err != nil {
				t.Fatal(err)
			}
			for _, issue := range model.Issues {
				if issue.Severity == "error" {
					t.Fatalf("instantiated %s architecture templates produced an error: %#v", language, issue)
				}
			}
		})
	}
}

func TestToudocuArchitectureGuidanceAndSemanticGate(t *testing.T) {
	for _, language := range []string{"en"} {
		guidance := readToudocuFile(t, filepath.Join("assets", "project-guidance", language+".md"))
		for _, expected := range []string{"architecture/overview.md", "FLOW-*", "CONTRACT", "REFERENCE", "RUNBOOK", "ADR", "MODULE"} {
			if !strings.Contains(guidance, expected) {
				t.Errorf("%s architecture guidance does not contain %q", language, expected)
			}
		}
	}

	gate := readToudocuFile(t, filepath.Join("references", "architecture-gate.md"))
	for i := 1; i <= 13; i++ {
		code := fmt.Sprintf("ARCH%03d", i)
		if strings.Count(gate, code) == 0 {
			t.Errorf("semantic gate must define %s", code)
		}
	}
	for _, expected := range []string{
		"Review the overview separately",
		"a transitive link is not",
		"any non-empty question text",
		"Punctuation, question",
	} {
		if !containsNormalized(gate, expected) {
			t.Errorf("semantic gate does not contain %q", expected)
		}
	}
}

func TestToudocuProjectGuidanceTemplates(t *testing.T) {
	const startMarker = "<!-- toudocu:project-guidance:start -->"
	const endMarker = "<!-- toudocu:project-guidance:end -->"

	for _, language := range []string{"en"} {
		content := readToudocuFile(t, filepath.Join("assets", "project-guidance", language+".md"))
		if strings.Count(content, startMarker) != 1 || strings.Count(content, endMarker) != 1 {
			t.Errorf("%s guidance must contain each managed marker exactly once", language)
		}
		if strings.Index(content, startMarker) >= strings.Index(content, endMarker) {
			t.Errorf("%s guidance markers are not ordered", language)
		}
		trimmed := strings.TrimSpace(content)
		if !strings.HasPrefix(trimmed, startMarker) || !strings.HasSuffix(trimmed, endMarker) {
			t.Errorf("%s guidance contains content outside the managed block", language)
		}
		for _, expected := range []string{"$toudocu", "TASK-*", "$toudocu init"} {
			if !strings.Contains(content, expected) {
				t.Errorf("%s guidance does not contain %q", language, expected)
			}
		}
	}

	en := readToudocuFile(t, filepath.Join("assets", "project-guidance", "en.md"))
	if !containsNormalized(en, "Do not create a task for every prompt") {
		t.Error("English guidance does not prevent per-prompt task creation")
	}
}

func TestToudocuTranslationContextIsolation(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	workflows := readToudocuFile(t, filepath.Join("references", "workflows.md"))
	refresh := readToudocuFile(t, filepath.Join("references", "refresh.md"))
	translate := readToudocuFile(t, filepath.Join("references", "translate.md"))
	initReference := readToudocuFile(t, filepath.Join("references", "init.md"))
	for _, expected := range []string{"$toudocu translate <target-locale> --from <source-locale>", "Source and target must differ", "never change the source root", "does not synchronize task status"} {
		if !containsNormalized(translate, expected) {
			t.Errorf("peer-locale translation workflow missing %q", expected)
		}
	}
	if !containsNormalized(skill, "Each configured locale root is an independent documentation and backlog source") {
		t.Error("skill misses peer-locale invariant")
	}
	if skill != "" {
		return
	}

	for name, content := range map[string]string{"workflows.md": workflows, "refresh.md": refresh} {
		for _, expected := range []string{"canonical documentation root", "translation root"} {
			if !containsNormalized(content, expected) {
				t.Errorf("%s does not isolate translations with %q", name, expected)
			}
		}
	}

	for _, expected := range []string{
		"Never use configured translation roots as canonical documentation or backlog context",
		"explicitly selected locale translation, check, find, build, run, or inspection operation",
	} {
		if !containsNormalized(skill, expected) {
			t.Errorf("SKILL.md does not contain translation invariant %q", expected)
		}
	}

	if !strings.Contains(refresh, "Never inventory a configured translation root") {
		t.Error("refresh workflow may inventory translations")
	}
	if !strings.Contains(refresh, "translation review requires an explicit locale-specific request") {
		t.Error("refresh workflow does not preserve the explicit locale exception")
	}
	if !strings.Contains(workflows, "Never read a translated `TASK-*` or `BUG-*` as task context") {
		t.Error("task workflow may read translated work items")
	}
	for _, expected := range []string{
		"one source/target pair at a time",
		"relative paths",
		"manifest source digests",
		"structural reports",
		"Preserve every `toudocu` annotation byte-for-byte",
		"canonical enum values",
		"Translate only reader-facing headings",
		"`documents[].type` and `documents[].status.kind`",
		"`effectiveCompleted`, `completionSource`",
		"TRANSLATION_SEMANTIC_MISMATCH",
		"$toudocu translate diff",
		"every configured translation profile",
		"--base HEAD --target working-tree",
		"staged, unstaged, and untracked canonical changes",
		"normalized locale order",
		"continue with the remaining locales",
		"TRANSLATION_PROFILES_EMPTY",
		"TRANSLATION_DIFF_UNAVAILABLE",
		"--translation-input",
		"regardless of `changes.includeTaskArtifacts`, `changes.includeAssets`, or arbitrary\n`changes.exclude`",
		"Translation never creates or\ncompletes a profile",
	} {
		if !containsNormalized(translate, expected) {
			t.Errorf("translation workflow does not minimize context with %q", expected)
		}
	}
	if !containsNormalized(initReference, "Use `assets/project-guidance/en.md` for every project locale") ||
		!containsNormalized(initReference, "isolate translation roots from ordinary work") {
		t.Error("init does not require translation isolation in installed guidance")
	}

	guidanceCases := map[string][]string{
		"en": {"only documentation and backlog source", "outside translation roots remain valid implementation evidence", "explicit `$toudocu translate <locale>`", "check, find, build, run, or inspect", "$toudocu translate diff", "Process locales one at a time", "do not add translation roots to ignore files"},
	}
	for language, expectedValues := range guidanceCases {
		guidance := readToudocuFile(t, filepath.Join("assets", "project-guidance", language+".md"))
		for _, expected := range expectedValues {
			if !containsNormalized(guidance, expected) {
				t.Errorf("%s guidance does not contain translation isolation scenario %q", language, expected)
			}
		}
	}
}

func TestToudocuInstructionConsistency(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	initReference := readToudocuFile(t, filepath.Join("references", "init.md"))
	translate := readToudocuFile(t, filepath.Join("references", "translate.md"))
	workflows := readToudocuFile(t, filepath.Join("references", "workflows.md"))
	documentModel := readToudocuFile(t, filepath.Join("references", "document-model.md"))

	for name, content := range map[string]string{
		"SKILL.md": skill, "init.md": initReference, "translate.md": translate, "workflows.md": workflows,
	} {
		for _, forbidden := range []string{"toudocu check ./docs", "toudocu changes ./docs", "toudocu task context TASK-AREA-001 ./docs"} {
			if strings.Contains(content, forbidden) {
				t.Errorf("%s hardcodes the fallback documentation root in an executable instruction: %q", name, forbidden)
			}
		}
	}
	for _, language := range []string{"en"} {
		guidance := readToudocuFile(t, filepath.Join("assets", "project-guidance", language+".md"))
		if strings.Contains(guidance, "docs/architecture/overview.md") {
			t.Errorf("%s guidance hardcodes the fallback documentation root", language)
		}
	}

	for _, expected := range []string{
		"Every project requires `index.md` and `architecture/overview.md`",
		"A missing\n`index.md` is a warning",
		"architecture overview\nis an error",
		"Requires canonical `architectureQuestion` metadata and a direct overview listing",
	} {
		if !containsNormalized(documentModel, expected) {
			t.Errorf("document-model.md misses architecture consistency statement %q", expected)
		}
	}
	for _, expected := range []string{
		"A repository-root `CHANGELOG.md`, when present, is the only special release journal",
		"Do not create a portal-specific duplicate inside the documentation root",
	} {
		if !containsNormalized(documentModel, expected) {
			t.Errorf("document-model.md misses changelog contract %q", expected)
		}
	}
	if strings.Contains(documentModel, "Only `index.md` is globally expected") || strings.Contains(documentModel, "Requires document type `Architecture`") {
		t.Fatal("document-model.md retains a contradictory architecture contract")
	}
	for _, expected := range []string{
		"`QUALITY` target when standards are declared",
		"Read-only unless `--report` writes JSON",
		"save, create, and roadmap-add actions can change canonical sources",
		"`--no-update-check`",
		"Do not add translation roots to `.gitignore` or global ignore files",
		"Use an initial read-only `check` only when establishing a baseline",
		"Scale the summary to the size and risk of the work",
		"what remains unverified or needs attention",
	} {
		if !containsNormalized(workflows, expected) {
			t.Errorf("workflows.md misses side-effect or verification contract %q", expected)
		}
	}
}

func TestToudocuTaskCreationThreshold(t *testing.T) {
	skill := readToudocuFile(t, "SKILL.md")
	workflows := readToudocuFile(t, filepath.Join("references", "workflows.md"))
	for name, content := range map[string]string{"SKILL.md": skill, "workflows.md": workflows} {
		for _, expected := range []string{"explicitly requires", "substantial", "Do not create"} {
			if !containsNormalized(content, expected) {
				t.Errorf("%s does not contain task threshold %q", name, expected)
			}
		}
	}
	for _, forbidden := range []string{
		"For a new request, search and create a neutral Draft",
		"For a new request, start with `search`, `task init`",
	} {
		if strings.Contains(skill, forbidden) || strings.Contains(workflows, forbidden) {
			t.Errorf("unconditional task workflow remains: %q", forbidden)
		}
	}
}

func TestToudocuMetadata(t *testing.T) {
	metadata := readToudocuFile(t, filepath.Join("agents", "openai.yaml"))
	for _, expected := range []string{
		`display_name: "Toudocu"`,
		`short_description: "Write and validate reliable, reader-first project documentation"`,
		`default_prompt: "Use $toudocu for this request. Select only the applicable workflow and references.`,
	} {
		if !strings.Contains(metadata, expected) {
			t.Errorf("openai.yaml does not contain %q", expected)
		}
	}
}

func writeSkillTemplate(t *testing.T, docs, language, templateName, destination string, replacements map[string]string) {
	t.Helper()
	rendered := readSkillTemplate(t, language, templateName)
	for placeholder, value := range replacements {
		rendered = strings.ReplaceAll(rendered, placeholder, value)
	}
	if strings.Contains(rendered, "{{") {
		t.Fatalf("unresolved placeholder in %s/%s:\n%s", language, templateName, rendered)
	}
	writeTestFile(t, docs, destination, rendered)
}

func readToudocuFile(t *testing.T, relativePath string) string {
	t.Helper()
	content, err := os.ReadFile(repositoryPath("skills", "toudocu", relativePath))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func readSkillTemplate(t *testing.T, language, templateName string) string {
	t.Helper()
	templatePath := repositoryPath("skills", "toudocu", "assets", "templates", language, templateName)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func containsNormalized(content, expected string) bool {
	return strings.Contains(strings.Join(strings.Fields(content), " "), strings.Join(strings.Fields(expected), " "))
}

func repositoryPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}
