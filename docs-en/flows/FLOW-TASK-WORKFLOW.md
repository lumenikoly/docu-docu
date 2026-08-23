<!-- toudocu
id: FLOW-TASK-WORKFLOW
module: MOD-CLI
useCase: UC-TASK-01, UC-TASK-02, UC-TASK-03
updated: 2026-08-23
-->

# FLOW-TASK-WORKFLOW: Work with a verifiable task

The diagram connects task preparation, safe context collection, and a separate,
explicitly authorized verification run.

## Process

```mermaid
flowchart TD
    Search["Find related documents: toudocu search"] --> Init["Create a draft: toudocu task init"]
    Init --> Split{"Does the work need decomposition into independent outcomes?"}
    Split -->|Yes| Children["Create child TASK-* items with Parent"]
    Split -->|No| Fill["Fill in the goal, boundaries, links, criteria, and verification commands"]
    Children --> Fill
    Fill --> Ready["Check completeness: toudocu task ready TASK-ID"]
    Ready --> Complete{"Is the contract complete?"}
    Complete -->|No| Fill
    Complete -->|Yes| Status["Manually change Draft to Ready"]
    Status --> Selected{"Has a task already been selected?"}
    Selected -->|No| Candidates["List task candidates: toudocu task candidates"]
    Candidates --> Choose["The implementer selects a task"]
    Choose --> Context["Collect context: toudocu task context TASK-ID"]
    Selected -->|Yes| Context
    Status -.->|Need a decomposition overview| Tree["Show the tree: toudocu task tree TASK-ID"]
    Tree --> Context
    Context --> Found{"Was exactly one task found?"}
    Found -->|No| ContextError["Return an error without running commands"]
    Found -->|Yes| Plan["Plan and perform the work outside Toudocu"]
    Plan --> DryRun["Explicitly inspect the plan: task verify --dry-run"]
    DryRun --> Run["After separate authorization: task verify --run"]
    Run --> Valid{"Is the verification contract valid?"}
    Valid -->|No| Blocked["Return blocked without running commands"]
    Valid -->|Yes| Commands["Run declared commands sequentially"]
    Commands --> Passed{"Did every command pass?"}
    Passed -->|Yes| Success["Return passed and code 0"]
    Passed -->|No| Failed["Return failed and code 1"]
```

## Important behavior

- `task context`, `task ready`, `task candidates`, and `task verify --dry-run`
  neither execute system commands nor change Markdown.
- `task candidates` computes contract completeness and dependency completion
  but does not choose the next task.
- `task verify --run` starts only after explicit authorization and runs only the
  trusted commands written in the selected task; it does not run child-task
  commands.
- One failed command does not hide the results of the others; a timeout stops
  that command's entire process tree.
- A person or agent—not Toudocu—judges the request's meaning, whether the links
  are appropriate, and whether acceptance criteria are satisfied.

## Related documents

- [UC-TASK-01: Get work task context](../use-cases/task-workflow.md)
- [UC-TASK-02: Perform work task checks](../use-cases/task-verify.md)
- [UC-TASK-03: Prepare a new work task](../use-cases/UC-TASK-03.md)
- [MOD-CLI: CLI and work-item operations](../modules/cli.md)
- [Work-item guide](../guides/work-items.md)
- [CLI contract](../contracts/cli.md)
