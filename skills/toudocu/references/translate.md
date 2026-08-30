# Translate one configured locale into another

Use this workflow only for an explicit request:

```text
$toudocu translate <target-locale> --from <source-locale> (--task <TASK-ID> | --base <ref> | --all-stale)
```

Normalize both locale keys as BCP-47-style locales and require both profiles
under `locales.*`. Source and target must differ. `project.defaultLocale` only
selects the default portal; it gives no locale authority over another.

Process exactly one source/target pair and one selection mode. Read the source
root for meaning and the target root for existing target-language text. Never
use a third locale as context and never change the source root.

- `--task <TASK-ID>` selects the source task and its task context.
- `--base <ref>` selects source changes relative to the Git ref.
- `--all-stale` selects stale target documents using existing translation
  bookkeeping.

Preserve stable IDs, commands, flags, paths, URLs, canonical enum values and
Toudocu annotations exactly. Translate reader-facing prose into the target
language. Write only the selected target root, validate that root, and report
files changed or skipped. Toudocu does not synchronize task status, dependencies
or other machine fields automatically; the team reviews cross-locale parity.

Legacy `.toudocu/translations/*.json` files may support stale selection in this
explicit workflow. They are not runtime configuration, establish no canonical
locale, and are never consulted by check, build, serve, Editor, Agent Console
or task mutations.
