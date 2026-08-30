package toudocu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func PrintHelp(w io.Writer) {
	_, _ = fmt.Fprintf(w, `Toudocu %s

Usage:
  toudocu COMMAND [options]

Commands:
  check       Validate source documentation without changing it
  build       Build a standalone read-only portal
  serve       Start the local portal, editor, and live rebuild
  changes     Show Git-backed documentation changes
  agent       Read a local request or save an agent response
  search      Search source Markdown documents
  scaffold    Create one typed document
  task        Operate the work-item lifecycle
  skill       Install the bundled AI skill for a supported host
  version     Show the version

For command options and side effects:
  toudocu COMMAND --help
  toudocu task --help
`, Version)
}

func PrintCommandHelp(w io.Writer, topic string) {
	help := map[string]string{
		"build": `Builds a standalone read-only portal and writes the output.

Usage:
  toudocu build [docs-dir] [-o DIR] [--clean] [--open] [--strict]
                [--exclude PATHS] [--stale-days N] [--repository-root DIR]
                [--repository-url URL] [--repository-ref REF]
                [--screen-map|--no-screen-map] [-t TITLE]

Example:
  toudocu build ./docs -o ./build/project-docs --clean

Side effects: writes output; --clean first removes only a validated safe output directory.`,
		"check": `Validates structure, links, IDs, and explicit relationships without changing files.

Usage:
  toudocu check [docs-dir] [--strict] [--format text|json]
                [--exclude PATHS] [--stale-days N] [--repository-root DIR]

Example:
  toudocu check ./docs --strict

Side effects: none. Without --strict, warnings do not change the exit code.`,
		"serve": `Builds the portal and starts a local HTTP/editor workspace with live rebuild.

Usage:
  toudocu serve [docs-dir] [-o DIR] [--host ADDRESS] [--port N]
                [--open] [--strict] [--exclude PATHS] [--stale-days N]
                [--repository-root DIR] [--screen-map|--no-screen-map]
                [--no-update-check] [-t TITLE]

Example:
  toudocu serve ./docs --host 127.0.0.1 --port 8080

Side effects: writes output and starts HTTP; explicit browser actions for save,
create, and roadmap add change the workspace. The first canonical portal request
checks the latest stable GitHub Release once; --no-update-check disables it.`,
		"changes": `Builds a read-only Git-backed documentation changes report.

Usage:
  toudocu changes [docs-dir] [--base REV|--branch-base REF]
                  [--target working-tree|index|HEAD|REV]
                  [--status STATUS] [--module ID] [--type TYPE]
                  [--permanent-only] [--include-assets|--translation-input]
                  [--repository-root DIR] [--format text|json|markdown] [-o FILE]
  toudocu changes file PATH [docs-dir] [same options]
Example:
  toudocu changes ./docs --base main --target working-tree --format markdown

--include-assets includes binary assets regardless of changes.includeAssets.
--translation-input includes reader-facing Markdown, work artifacts, and assets,
ignoring changes.exclude except generated/** and cache/** inside the docs root.

Side effects: reads Git and the workspace; -o writes the explicitly selected report.`,
		"agent": `Reads the local Toudocu queue or saves a development-agent response.

Usage:
  toudocu agent next [--repository-root DIR] --json
  toudocu agent respond [--input response.json] [--repository-root DIR] [--json]

Without --input, respond reads JSON from standard input. These commands change
only local user state outside the repository and do not run a language model.`,
		"changes-file": `Shows details for one changed path without changing files.

Usage:
  toudocu changes file PATH [docs-dir] [--base REV|--branch-base REF]
                       [--target working-tree|index|HEAD|REV]
                       [--include-assets|--translation-input]
                       [--repository-root DIR]
                       [--format text|json|markdown] [-o FILE]

Side effects: reads Git and the workspace; -o writes the explicitly selected report file.`,
		"search": `Searches current source Markdown without changing files.

Usage:
  toudocu search "QUERY" [docs-dir] [--limit N] [--format text|json]

Example:
  toudocu search "task workflow" ./docs --format json`,
		"scaffold": `Atomically creates one typed Markdown file.

Usage:
  toudocu scaffold module|use-case|flow|screen|decision|standard|runbook ID
                   [docs-dir] --title TITLE [--lang en|ru] [--format text|json]

Without --lang, the language comes from .toudocu/config.yml; the fallback is en.
Example:
  toudocu scaffold module MOD-CLI ./docs --title "CLI"`,
		"task": `Operates the work-item lifecycle.

Usage:
  toudocu task init|ready|candidates|context|verify|archive|restore|changes|tree ...

For operation options:
  toudocu task OPERATION --help`,
		"task-init": `Atomically creates a new Draft TASK-* or BUG-*.

Usage:
  toudocu task init [docs-dir] --area AREA --title TITLE --type TYPE [--parent TASK-ID]
                    [--lang en|ru] [--format text|json]

TYPE: Feature, Bug, Maintenance, Documentation, or Research.
Without --lang, the language comes from .toudocu/config.yml; the fallback is en.`,
		"task-ready": `Validates a Draft or Ready contract without changing files.

Usage:
  toudocu task ready TASK-ID [docs-dir] [--strict] [--format text|json]`,
		"task-candidates": `Lists Draft and Ready work candidates without changing files.

Usage:
  toudocu task candidates [docs-dir] [--parent TASK-ID] [--strict] [--format text|json]

Without --parent, candidates come from all active work items. With --parent,
only descendants of that TASK-* are included.`,
		"task-context": `Returns compact read-only context for a Ready+ task.

Usage:
  toudocu task context TASK-ID [docs-dir] [--repository-root DIR] [--format text|json]`,
		"task-tree": `Shows a read-only TASK-* decomposition tree.

Usage:
  toudocu task tree TASK-ID [docs-dir] [--repository-root DIR] [--format text|json]`,
		"task-verify": `Plans or runs trusted task verification commands.

Usage:
  toudocu task verify TASK-ID [docs-dir] (--dry-run|--run)
                      [--target TARGET] [--report FILE] [--timeout DURATION]
                      [--repository-root DIR] [--format text|json]

--dry-run does not run commands. --report writes a JSON file in either mode;
--run executes the task commands.`,
		"task-archive": `Moves a valid Done or Cancelled task to work/archive/YYYY without overwriting.

Usage:
  toudocu task archive TASK-ID [docs-dir] [--repository-root DIR] [--format text|json]`,
		"task-restore": `Moves an archived task back to work/ without overwriting.

Usage:
  toudocu task restore TASK-ID [docs-dir] [--repository-root DIR] [--format text|json]`,
		"task-changes": `Builds one task-scoped read-only changes and impact report.

Usage:
  toudocu task changes TASK-ID [docs-dir] [--base REV|--branch-base REF]
                       [--target working-tree|index|HEAD|REV]
                       [--include-assets|--translation-input]
                       [--repository-root DIR]
                       [--tree] [--format text|json|markdown] [-o FILE]

Side effects: reads Git and the workspace; -o writes the explicitly selected
report file. --tree includes the selected task and all its descendants.`,
		"skill": `Manages the bundled offline Toudocu AI-skill package.

Usage:
  toudocu skill install|status|update|uninstall
                  [--agent auto|codex|claude-code|copilot|all]
                  [--scope project|user] [--repository-root DIR]

--repository-root is available only for project scope. status changes nothing.`,
		"version": "Shows the Toudocu version without side effects.\n\nUsage:\n  toudocu version",
	}
	if text, ok := help[topic]; ok {
		_, _ = fmt.Fprintln(w, text)
		return
	}
	PrintHelp(w)
}

func helpTopic(argv []string) (string, bool) {
	hasHelp := false
	for _, arg := range argv {
		if arg == "-h" || arg == "--help" {
			hasHelp = true
		}
	}
	if len(argv) > 0 && argv[0] == "help" {
		hasHelp = true
		argv = argv[1:]
	}
	if !hasHelp {
		return "", false
	}
	words := []string{}
	for _, arg := range argv {
		if arg == "-h" || arg == "--help" || strings.HasPrefix(arg, "-") {
			continue
		}
		words = append(words, arg)
	}
	if len(words) == 0 {
		return "", true
	}
	if words[0] == "changes" && len(words) > 1 && words[1] == "file" {
		return "changes-file", true
	}
	if words[0] == "task" && len(words) > 1 {
		return "task-" + words[1], true
	}
	return words[0], true
}

func takeArgValue(args []string, index *int, option string) (string, error) {
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "-") {
		return "", fmt.Errorf("option %s requires a value", option)
	}
	*index++
	return args[*index], nil
}

// ParseArguments parses explicit subcommands.
func ParseArguments(argv []string) (Options, bool, bool, error) {
	options := Options{StaleDays: 90, RepositoryRef: "main", Format: "text", Timeout: 10 * time.Minute, Host: "127.0.0.1", Port: 8080, Limit: 20}
	help, version := false, false
	timeoutSpecified, hostSpecified, portSpecified := false, false, false
	titleSpecified, languageSpecified, limitSpecified, outputSpecified := false, false, false, false
	changesOptionSpecified := false
	screenMapOption := ""
	args := append([]string{}, argv...)
	if len(args) > 0 {
		switch args[0] {
		case "build", "check", "serve":
			options.Command = args[0]
			args = args[1:]
		case "search":
			if len(args) < 2 {
				return options, false, false, fmt.Errorf("usage: toudocu search \"<query>\" [docs-directory]")
			}
			options.Command, options.Query = "search", args[1]
			args = args[2:]
		case "changes":
			options.Command = "changes"
			args = args[1:]
			if len(args) > 0 && args[0] == "file" {
				if len(args) < 2 {
					return options, false, false, fmt.Errorf("usage: toudocu changes file PATH [docs-directory]")
				}
				options.Command, options.ChangeFile = "changes-file", filepath.ToSlash(args[1])
				args = args[2:]
			}
		case "scaffold":
			if len(args) < 3 {
				return options, false, false, fmt.Errorf("usage: toudocu scaffold module|use-case|flow|screen|decision|standard|runbook ID [docs-directory]")
			}
			options.Command, options.EntityKind, options.EntityID = "scaffold", args[1], args[2]
			args = args[3:]
		case "init", "refresh":
			return options, false, false, fmt.Errorf("unknown command: %s", args[0])
		case "version":
			version = true
			args = args[1:]
		case "help":
			help = true
			args = args[1:]
		case "task":
			if len(args) < 2 {
				return options, false, false, fmt.Errorf("usage: toudocu task init|ready|candidates|context|verify|archive|restore")
			}
			switch args[1] {
			case "init":
				options.Command = "task-init"
				args = args[2:]
				goto parseOptions
			case "candidates":
				options.Command = "task-candidates"
				args = args[2:]
				goto parseOptions
			case "ready", "context", "tree", "verify", "archive", "restore", "changes":
				if len(args) < 3 {
					return options, false, false, fmt.Errorf("task %s requires TASK-ID", args[1])
				}
				options.Command = "task-" + args[1]
			default:
				return options, false, false, fmt.Errorf("usage: toudocu task init|ready|candidates|context|verify|archive|restore")
			}
			options.TaskID = args[2]
			args = args[3:]
		default:
			if !strings.HasPrefix(args[0], "-") {
				return options, false, false, fmt.Errorf("unknown command: %s", args[0])
			}
		}
	}
	if options.Command == "" && !help && !version {
		return options, false, false, fmt.Errorf("command required; use toudocu --help")
	}
parseOptions:
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			if i+1 < len(args) && options.InputDirectory == "" {
				options.InputDirectory = args[i+1]
			}
			i = len(args)
		case arg == "-h" || arg == "--help":
			help = true
		case arg == "-v" || arg == "--version":
			version = true
		case arg == "-o" || arg == "--output":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
				options.ChangeOutput = v
			} else {
				options.OutputDirectory = v
			}
			outputSpecified = true
		case strings.HasPrefix(arg, "--output="):
			if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
				options.ChangeOutput = strings.TrimPrefix(arg, "--output=")
			} else {
				options.OutputDirectory = strings.TrimPrefix(arg, "--output=")
			}
			outputSpecified = true
		case arg == "-t" || arg == "--title":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.Title = v
			titleSpecified = true
		case strings.HasPrefix(arg, "--title="):
			options.Title = strings.TrimPrefix(arg, "--title=")
			titleSpecified = true
		case arg == "--area":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.Area = v
		case strings.HasPrefix(arg, "--area="):
			options.Area = strings.TrimPrefix(arg, "--area=")
		case arg == "--parent":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.ParentTaskID = v
		case strings.HasPrefix(arg, "--parent="):
			options.ParentTaskID = strings.TrimPrefix(arg, "--parent=")
		case arg == "--tree":
			options.ChangeTaskTree = true
		case arg == "--type":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
				options.ChangeEntityType = v
			} else {
				options.TaskType = v
			}
		case strings.HasPrefix(arg, "--type="):
			if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
				options.ChangeEntityType = strings.TrimPrefix(arg, "--type=")
			} else {
				options.TaskType = strings.TrimPrefix(arg, "--type=")
			}
		case arg == "--base":
			changesOptionSpecified = true
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.ChangeBase = v
		case strings.HasPrefix(arg, "--base="):
			changesOptionSpecified = true
			options.ChangeBase = strings.TrimPrefix(arg, "--base=")
		case arg == "--branch-base":
			changesOptionSpecified = true
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.ChangeBranchBase = v
		case strings.HasPrefix(arg, "--branch-base="):
			changesOptionSpecified = true
			options.ChangeBranchBase = strings.TrimPrefix(arg, "--branch-base=")
		case arg == "--status":
			changesOptionSpecified = true
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.ChangeStatus = v
		case strings.HasPrefix(arg, "--status="):
			changesOptionSpecified = true
			options.ChangeStatus = strings.TrimPrefix(arg, "--status=")
		case arg == "--module":
			changesOptionSpecified = true
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.ChangeModule = v
		case strings.HasPrefix(arg, "--module="):
			changesOptionSpecified = true
			options.ChangeModule = strings.TrimPrefix(arg, "--module=")
		case arg == "--permanent-only":
			changesOptionSpecified = true
			options.ChangePermanentOnly = true
		case arg == "--include-assets":
			changesOptionSpecified = true
			options.ChangeForceIncludeAssets = true
		case arg == "--translation-input":
			changesOptionSpecified = true
			options.ChangeTranslationInput = true
		case arg == "--lang":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.Language = v
			languageSpecified = true
		case strings.HasPrefix(arg, "--lang="):
			options.Language = strings.TrimPrefix(arg, "--lang=")
			languageSpecified = true
		case arg == "--limit":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 100 {
				return options, false, false, fmt.Errorf("--limit must be a number from 1 to 100")
			}
			options.Limit = n
			limitSpecified = true
		case strings.HasPrefix(arg, "--limit="):
			n, e := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if e != nil || n < 1 || n > 100 {
				return options, false, false, fmt.Errorf("--limit must be a number from 1 to 100")
			}
			options.Limit = n
			limitSpecified = true
		case arg == "--dry-run":
			if options.VerifyMode != "" {
				return options, false, false, fmt.Errorf("--dry-run and --run cannot be used together")
			}
			options.VerifyMode = "dry-run"
		case arg == "--run":
			if options.VerifyMode != "" {
				return options, false, false, fmt.Errorf("--dry-run and --run cannot be used together")
			}
			options.VerifyMode = "run"
		case arg == "--target":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
				options.ChangeTarget = v
			} else {
				options.Target = strings.ToUpper(v)
			}
		case strings.HasPrefix(arg, "--target="):
			if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
				options.ChangeTarget = strings.TrimPrefix(arg, "--target=")
			} else {
				options.Target = strings.ToUpper(strings.TrimPrefix(arg, "--target="))
			}
		case arg == "--exclude":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.Excludes = append(options.Excludes, splitCSV(v)...)
		case strings.HasPrefix(arg, "--exclude="):
			options.Excludes = append(options.Excludes, splitCSV(strings.TrimPrefix(arg, "--exclude="))...)
		case arg == "--stale-days":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			n, e := strconv.Atoi(v)
			if e != nil || n < 0 {
				return options, false, false, fmt.Errorf("--stale-days must be a non-negative number")
			}
			options.StaleDays = n
		case strings.HasPrefix(arg, "--stale-days="):
			n, e := strconv.Atoi(strings.TrimPrefix(arg, "--stale-days="))
			if e != nil || n < 0 {
				return options, false, false, fmt.Errorf("--stale-days must be a non-negative number")
			}
			options.StaleDays = n
		case arg == "--repository-root":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.RepositoryRoot = v
		case strings.HasPrefix(arg, "--repository-root="):
			options.RepositoryRoot = strings.TrimPrefix(arg, "--repository-root=")
		case arg == "--repository-url":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.RepositoryURL = v
		case strings.HasPrefix(arg, "--repository-url="):
			options.RepositoryURL = strings.TrimPrefix(arg, "--repository-url=")
		case arg == "--repository-ref":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.RepositoryRef = v
		case strings.HasPrefix(arg, "--repository-ref="):
			options.RepositoryRef = strings.TrimPrefix(arg, "--repository-ref=")
		case arg == "--format":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.Format = v
		case strings.HasPrefix(arg, "--format="):
			options.Format = strings.TrimPrefix(arg, "--format=")
		case arg == "--report":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.ReportPath = v
		case strings.HasPrefix(arg, "--report="):
			options.ReportPath = strings.TrimPrefix(arg, "--report=")
		case arg == "--timeout":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			duration, e := time.ParseDuration(v)
			if e != nil || duration <= 0 {
				return options, false, false, fmt.Errorf("--timeout must be a positive duration, for example 10m")
			}
			options.Timeout = duration
			timeoutSpecified = true
		case strings.HasPrefix(arg, "--timeout="):
			duration, e := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if e != nil || duration <= 0 {
				return options, false, false, fmt.Errorf("--timeout must be a positive duration, for example 10m")
			}
			options.Timeout = duration
			timeoutSpecified = true
		case arg == "--host":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			options.Host = v
			hostSpecified = true
		case strings.HasPrefix(arg, "--host="):
			options.Host = strings.TrimPrefix(arg, "--host=")
			hostSpecified = true
		case arg == "--port":
			v, e := takeArgValue(args, &i, arg)
			if e != nil {
				return options, false, false, e
			}
			port, e := strconv.Atoi(v)
			if e != nil || port < 1 || port > 65535 {
				return options, false, false, fmt.Errorf("--port must be a number from 1 to 65535")
			}
			options.Port = port
			portSpecified = true
		case strings.HasPrefix(arg, "--port="):
			port, e := strconv.Atoi(strings.TrimPrefix(arg, "--port="))
			if e != nil || port < 1 || port > 65535 {
				return options, false, false, fmt.Errorf("--port must be a number from 1 to 65535")
			}
			options.Port = port
			portSpecified = true
		case arg == "--clean":
			options.Clean = true
		case arg == "--open":
			options.Open = true
		case arg == "--strict":
			options.Strict = true
		case arg == "--screen-map":
			if screenMapOption == "off" {
				return options, false, false, fmt.Errorf("--screen-map and --no-screen-map cannot be used together")
			}
			screenMapOption = "on"
			options.NoScreenMap = false
		case arg == "--no-screen-map":
			if screenMapOption == "on" {
				return options, false, false, fmt.Errorf("--screen-map and --no-screen-map cannot be used together")
			}
			screenMapOption = "off"
			options.NoScreenMap = true
		case arg == "--no-update-check":
			options.NoUpdateCheck = true
		case strings.HasPrefix(arg, "-"):
			return options, false, false, fmt.Errorf("unknown option: %s", arg)
		default:
			if options.InputDirectory == "" {
				options.InputDirectory = arg
			} else {
				return options, false, false, fmt.Errorf("unexpected positional argument: %s", arg)
			}
		}
	}
	if options.InputDirectory == "" {
		options.InputDirectory = "./docs"
	}
	input, err := filepath.Abs(options.InputDirectory)
	if err != nil {
		return options, false, false, err
	}
	options.InputDirectory = input
	if options.OutputDirectory == "" {
		options.OutputDirectory = filepath.Join(filepath.Dir(input), "project-docs")
	}
	output, err := filepath.Abs(options.OutputDirectory)
	if err != nil {
		return options, false, false, err
	}
	options.OutputDirectory = output
	if options.RepositoryRoot == "" {
		options.RepositoryRoot = filepath.Dir(input)
	}
	repo, err := filepath.Abs(options.RepositoryRoot)
	if err != nil {
		return options, false, false, err
	}
	options.RepositoryRoot = repo
	if !languageSpecified && (options.Command == "task-init" || options.Command == "scaffold") {
		options.Language = configuredScaffoldLanguage(options.RepositoryRoot, options.InputDirectory)
	}
	options.RepositoryURL = strings.TrimRight(options.RepositoryURL, "/")
	if options.RepositoryURL != "" && !strings.HasPrefix(strings.ToLower(options.RepositoryURL), "http://") && !strings.HasPrefix(strings.ToLower(options.RepositoryURL), "https://") {
		return options, false, false, fmt.Errorf("--repository-url must be an HTTP(S) URL")
	}
	if strings.TrimSpace(options.RepositoryRef) == "" {
		return options, false, false, fmt.Errorf("--repository-ref cannot be empty")
	}
	markdownChanges := options.Format == "markdown" && (options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes")
	if options.Format != "text" && options.Format != "json" && !markdownChanges {
		return options, false, false, fmt.Errorf("--format must be text, json, or markdown for changes")
	}
	if strings.TrimSpace(options.Host) == "" {
		return options, false, false, fmt.Errorf("--host cannot be empty")
	}
	if options.Command != "serve" && (hostSpecified || portSpecified) {
		return options, false, false, fmt.Errorf("--host and --port are available only for serve")
	}
	if screenMapOption != "" && options.Command != "build" && options.Command != "serve" {
		return options, false, false, fmt.Errorf("--screen-map and --no-screen-map are available only for build and serve")
	}
	if options.NoUpdateCheck && options.Command != "serve" {
		return options, false, false, fmt.Errorf("--no-update-check is available only for serve")
	}
	if options.Command == "task-ready" || options.Command == "task-context" || options.Command == "task-tree" || options.Command == "task-verify" || options.Command == "task-changes" ||
		options.Command == "task-archive" || options.Command == "task-restore" {
		if !taskIDRE.MatchString(options.TaskID) {
			return options, false, false, fmt.Errorf("work-item identifier must have the form TASK-AREA-NNN or BUG-AREA-NNN")
		}
	}
	if options.Command == "task-candidates" && options.ParentTaskID != "" && (!taskIDRE.MatchString(options.ParentTaskID) || !strings.HasPrefix(options.ParentTaskID, "TASK-")) {
		return options, false, false, fmt.Errorf("--parent must have the form TASK-AREA-NNN")
	}
	if options.Command == "task-verify" {
		if options.ReportPath != "" {
			report, err := filepath.Abs(options.ReportPath)
			if err != nil {
				return options, false, false, err
			}
			if !strings.EqualFold(filepath.Ext(report), ".json") {
				return options, false, false, fmt.Errorf("--report must point to a JSON file")
			}
			resolvedInput, err := resolvePathForSafety(options.InputDirectory)
			if err != nil {
				return options, false, false, err
			}
			resolvedReport, err := resolvePathForSafety(report)
			if err != nil {
				return options, false, false, err
			}
			if ensureInside(options.InputDirectory, report) || ensureInside(resolvedInput, resolvedReport) {
				return options, false, false, fmt.Errorf("--report cannot overwrite the source documentation directory")
			}
			if info, err := os.Stat(report); err == nil && info.IsDir() {
				return options, false, false, fmt.Errorf("--report must point to a file, not a directory")
			}
			options.ReportPath = report
		}
		if options.VerifyMode == "" {
			return options, false, false, fmt.Errorf("task verify requires exactly one mode: --dry-run or --run")
		}
	} else if options.ReportPath != "" || timeoutSpecified || options.VerifyMode != "" || options.Target != "" {
		return options, false, false, fmt.Errorf("--dry-run, --run, --target, --report, and --timeout are available only for task verify")
	}
	if options.Command == "task-init" {
		if options.Area == "" || !titleSpecified || strings.TrimSpace(options.Title) == "" || options.TaskType == "" {
			return options, false, false, fmt.Errorf("task init requires --area, --title, and --type")
		}
		if !taskAreaRE.MatchString(options.Area) {
			return options, false, false, fmt.Errorf("--area must contain A-Z, 0-9, and hyphens and start with a letter")
		}
		if !validTaskInitType(options.TaskType) {
			return options, false, false, fmt.Errorf("--type must be Feature, Bug, Maintenance, Documentation, or Research")
		}
	}
	if options.Command == "scaffold" && (!titleSpecified || strings.TrimSpace(options.Title) == "") {
		return options, false, false, fmt.Errorf("scaffold requires --title")
	}
	if (options.Command == "task-init" || options.Command == "scaffold") && strings.ContainsAny(options.Title, "\r\n") {
		return options, false, false, fmt.Errorf("--title must be a single line")
	}
	if options.Command == "scaffold" && !validScaffoldID(options.EntityKind, options.EntityID) {
		return options, false, false, fmt.Errorf("invalid %s ID: %s", options.EntityKind, options.EntityID)
	}
	if options.Command == "search" && len(searchWords(options.Query)) == 0 {
		return options, false, false, fmt.Errorf("search query cannot be empty")
	}
	if (options.Command == "task-init" || options.Command == "scaffold") && options.Language != "en" && options.Language != "ru" {
		return options, false, false, fmt.Errorf("--lang must be en or ru")
	}
	if languageSpecified && options.Command != "task-init" && options.Command != "scaffold" {
		return options, false, false, fmt.Errorf("--lang is available only for task init and scaffold")
	}
	if limitSpecified && options.Command != "search" {
		return options, false, false, fmt.Errorf("--limit is available only for search")
	}
	if options.Area != "" && options.Command != "task-init" || options.TaskType != "" && options.Command != "task-init" {
		return options, false, false, fmt.Errorf("--area and --type are available only for task init")
	}
	if options.ParentTaskID != "" && options.Command != "task-init" && options.Command != "task-candidates" {
		return options, false, false, fmt.Errorf("--parent is available only for task init and task candidates")
	}
	if options.ChangeTaskTree && options.Command != "task-changes" {
		return options, false, false, fmt.Errorf("--tree is available only for task changes")
	}
	if options.ChangeTaskTree && !strings.HasPrefix(options.TaskID, "TASK-") {
		return options, false, false, fmt.Errorf("--tree is available only for TASK-* work items")
	}
	if titleSpecified && options.Command != "build" && options.Command != "serve" && options.Command != "task-init" && options.Command != "scaffold" {
		return options, false, false, fmt.Errorf("--title is not available for this command")
	}
	if outputSpecified && options.Command != "build" && options.Command != "serve" && options.Command != "changes" && options.Command != "changes-file" && options.Command != "task-changes" {
		return options, false, false, fmt.Errorf("--output is available only for build and serve")
	}
	if (options.Clean || options.Open) && options.Command != "build" && options.Command != "serve" {
		return options, false, false, fmt.Errorf("--clean and --open are available only for build and serve")
	}
	if options.Strict && options.Command != "build" && options.Command != "check" && options.Command != "serve" && options.Command != "task-ready" && options.Command != "task-candidates" {
		return options, false, false, fmt.Errorf("--strict is not available for this command")
	}
	isChangesCommand := options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes"
	if changesOptionSpecified && !isChangesCommand {
		return options, false, false, fmt.Errorf("--base, --branch-base, --status, --module, --permanent-only, --include-assets, and --translation-input are available only for changes")
	}
	if isChangesCommand {
		if options.ChangeTranslationInput && options.ChangePermanentOnly {
			return options, false, false, fmt.Errorf("--translation-input and --permanent-only cannot be used together")
		}
		if options.ChangeBranchBase != "" && options.ChangeBase != "" {
			return options, false, false, fmt.Errorf("--base and --branch-base cannot be used together")
		}
		if options.ChangeOutput != "" {
			absolute, err := filepath.Abs(options.ChangeOutput)
			if err != nil {
				return options, false, false, err
			}
			options.ChangeOutput = absolute
		}
		if options.Command == "task-changes" {
			options.ChangeTaskID = options.TaskID
		}
	}
	return options, help, version, nil
}

func splitCSV(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func configuredScaffoldLanguage(repositoryRoot, inputRoot string) string {
	data, err := os.ReadFile(filepath.Join(repositoryRoot, ".toudocu", "config.yml"))
	if err != nil {
		return "en"
	}
	config, err := parseSiteConfig(data)
	if err != nil {
		return "en"
	}
	if absolute, absErr := filepath.Abs(inputRoot); absErr == nil {
		_, _ = selectLocaleProfile(&config, repositoryRoot, absolute)
	}
	language := strings.ToLower(strings.Split(config.Project.Locale, "-")[0])
	if language == "ru" || language == "en" {
		return language
	}
	return "en"
}

func openGeneratedSite(file string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{file}
	case "windows":
		command = "cmd"
		args = []string{"/c", "start", "", file}
	default:
		command = "xdg-open"
		args = []string{file}
	}
	return exec.Command(command, args...).Start()
}

func printCheckText(w io.Writer, model *Model) {
	_, _ = fmt.Fprintf(w, "Documents: %d\nWarnings: %d\nErrors: %d\n", model.Stats.Documents, model.Stats.Warnings, model.Stats.Errors)
	for _, issue := range model.Issues {
		if issue.Code == "DOCS_MIGRATION_REQUIRED" {
			_, _ = fmt.Fprintf(w, "\n%s\n\nMigration: %s\nFile: %s\n", issue.Code, issue.Migration, issue.DocumentPath)
			continue
		}
		location := issue.DocumentPath
		if issue.Line > 0 {
			location += fmt.Sprintf(":%d", issue.Line)
		}
		_, _ = fmt.Fprintf(w, "[%s] %s %s — %s\n", strings.ToUpper(issue.Severity), issue.Code, location, issue.Message)
	}
}

func hasDocumentationVersionIssue(model *Model) bool {
	for _, issue := range model.Issues {
		if issue.Code == "DOCS_MIGRATION_REQUIRED" || issue.Code == "DOCUMENTATION_VERSION_UNSUPPORTED" {
			return true
		}
	}
	return false
}

// RunCLI executes one command and returns a process exit code.
func RunCLI(argv []string, stdout, stderr io.Writer) int {
	if code := runAgentCLI(argv, os.Stdin, stdout, stderr); code >= 0 {
		return code
	}
	if topic, ok := helpTopic(argv); ok {
		PrintCommandHelp(stdout, topic)
		return 0
	}
	if len(argv) == 0 {
		PrintHelp(stdout)
		return 0
	}
	if argv[0] == "skill" {
		return runSkillCLI(argv, strings.NewReader(""), stdout, stderr, false)
	}
	options, help, version, err := ParseArguments(argv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "Error:", err)
		if len(argv) > 0 && (argv[0] == "changes" || (argv[0] == "task" && len(argv) > 1 && argv[1] == "changes")) {
			return 2
		}
		return 1
	}
	if help {
		PrintHelp(stdout)
		return 0
	}
	if version {
		_, _ = fmt.Fprintln(stdout, Version)
		return 0
	}
	if options.Command == "task-init" {
		report, err := InitTask(options)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			_, _ = fmt.Fprintf(stdout, "Created task %s: %s\n", report.ID, report.Path)
		}
		return 0
	}
	if options.Command == "scaffold" {
		report, err := Scaffold(options)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			_, _ = fmt.Fprintf(stdout, "Created %s %s: %s\n", report.EntityType, report.ID, report.Path)
		}
		return 0
	}
	if options.Command == "serve" {
		if err := serveDocumentation(options, stdout, stderr); err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		return 0
	}
	if options.Command == "changes" || options.Command == "changes-file" || options.Command == "task-changes" {
		report, err := BuildDocumentationChanges(options)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			var failure *changeFailure
			if errors.As(err, &failure) {
				return failure.Code
			}
			return 4
		}
		filterDocumentationChanges(report, options)
		if err := outputChangesReport(options, report, stdout); err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 4
		}
		for _, diagnostic := range report.Diagnostics {
			if diagnostic.Severity == "error" {
				return 1
			}
		}
		if report.TaskImpact != nil {
			for _, diagnostic := range report.TaskImpact.Diagnostics {
				if diagnostic.Severity == "error" {
					return 1
				}
			}
		}
		return 0
	}
	model, err := BuildDocumentationModel(options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if hasDocumentationVersionIssue(model) {
		if options.Format == "json" {
			data, _ := json.MarshalIndent(BuildReport(model), "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printCheckText(stdout, model)
		}
		return 1
	}
	if options.Command == "search" {
		report, err := SearchDocumentation(model, options.Query, options.Limit)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printSearchText(stdout, report)
		}
		return 0
	}
	if options.Command == "task-archive" || options.Command == "task-restore" {
		operation := strings.TrimPrefix(options.Command, "task-")
		report, err := MoveTask(model, options, operation)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, marshalErr := json.MarshalIndent(report, "", "  ")
			if marshalErr != nil {
				_, _ = fmt.Fprintln(stderr, "Error:", marshalErr)
				return 1
			}
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printTaskMoveText(stdout, report)
		}
		if report.Status == "archived" || report.Status == "restored" {
			return 0
		}
		return 1
	}
	if options.Command == "task-verify" {
		report := executeTaskVerify(model, options, stdout, stderr, osCommandRunner{})
		data, marshalErr := marshalTaskVerifyReport(report)
		if marshalErr != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", marshalErr)
			return 1
		}
		reportWriteFailed := false
		if options.ReportPath != "" {
			if err := writeReportAtomically(options.ReportPath, data); err != nil {
				_, _ = fmt.Fprintln(stderr, "Failed to save report:", err)
				reportWriteFailed = true
			}
		}
		if options.Format == "json" {
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printTaskVerifyText(stdout, report)
		}
		if (report.Status != "passed" && report.Status != "planned") || reportWriteFailed {
			return 1
		}
		return 0
	}
	if options.Command == "task-ready" {
		report := BuildTaskReady(model, options.TaskID, options.Strict)
		if options.Format == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printTaskReadyText(stdout, report)
		}
		if report.Status == "contract_ready" || report.Status == "ready" {
			return 0
		}
		return 1
	}
	if options.Command == "task-candidates" {
		report, err := BuildTaskCandidates(model, options.ParentTaskID, options.Strict)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, _ := json.MarshalIndent(report, "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printTaskCandidatesText(stdout, report)
		}
		return 0
	}
	if options.Command == "task-context" {
		report, err := BuildTaskContext(model, options.TaskID)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, marshalErr := json.MarshalIndent(report, "", "  ")
			if marshalErr != nil {
				_, _ = fmt.Fprintln(stderr, "Error:", marshalErr)
				return 1
			}
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printTaskContextText(stdout, report)
		}
		return 0
	}
	if options.Command == "task-tree" {
		report, err := BuildTaskTree(model, options.TaskID)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		if options.Format == "json" {
			data, marshalErr := json.MarshalIndent(report, "", "  ")
			if marshalErr != nil {
				_, _ = fmt.Fprintln(stderr, "Error:", marshalErr)
				return 1
			}
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printTaskTreeText(stdout, report)
		}
		return 0
	}
	if options.Command == "check" {
		if options.Format == "json" {
			data, _ := json.MarshalIndent(BuildReport(model), "", "  ")
			_, _ = fmt.Fprintln(stdout, string(data))
		} else {
			printCheckText(stdout, model)
		}
		if model.Stats.Errors > 0 || (options.Strict && model.Stats.Warnings > 0) {
			return 1
		}
		return 0
	}
	result, err := GenerateSite(model, options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "\nDocumentation built.\nDirectory:      %s\nPages:          %d\nDocuments:      %d\nRoadmap tasks:  %d\nCompleted:      %d\nWarnings:       %d\nErrors:         %d\nHome:           %s\n", result.OutputDirectory, result.Pages, model.Stats.Documents, model.Stats.TotalTasks, model.Stats.CompletedTasks, model.Stats.Warnings, model.Stats.Errors, filepath.Join(result.OutputDirectory, "index.html"))
	if options.Open {
		if err := openGeneratedSite(filepath.Join(result.OutputDirectory, "index.html")); err != nil {
			_, _ = fmt.Fprintln(stderr, "Failed to open the browser automatically:", err)
		}
	}
	if model.Stats.Errors > 0 || (options.Strict && model.Stats.Warnings > 0) {
		return 1
	}
	return 0
}

func Main() {
	if len(os.Args) > 1 && os.Args[1] == "skill" && !skillHelpRequested(os.Args[2:]) {
		os.Exit(runSkillCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, stdioInteractive(os.Stdin, os.Stdout)))
	}
	os.Exit(RunCLI(os.Args[1:], os.Stdout, os.Stderr))
}
