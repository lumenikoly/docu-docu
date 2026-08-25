<!-- toudocu
id: SC-SITE-DOCUMENT
status: done
screenKind: page
module: MOD-SITE
preview: ../assets/screens/site-document.png
updated: 2026-08-12
-->

# SC-SITE-DOCUMENT: Document Page


This is the ordinary page for one Markdown document. Its URL comes from the
source path, so this document does not declare one shared route pattern.

In side navigation, work items repeat their hierarchy: descendants appear under
their parent in a collapsible list. When a nested task is open, its branch stays
expanded.

## Documentation discussions

In the main `serve`, a context menu appears after the user selects text in any
meaningful part of the document: a heading, properties, or main content. It can
copy the selected text; copy the document title, path, and selection; or add a
question to a local discussion. The question form shows the selection, and the
user chooses a question or change request and enters a message.

When the selection maps to source Markdown, the discussion receives an exact
range. An ambiguous selection produces a clear error instead of losing context.
After saving, the shared project discussion panel opens on the right, so the
user does not need to visit Changes.

The “Discussions · N” button appears on the right side of the header on every
page of the main `serve`, and the count covers every open project thread. The
panel shows all project threads, open threads first. On a canonical document
page, the user can create a question without selecting text. On a page without
a current canonical document, this action is hidden while replies, editing,
closing, and deletion of existing threads remain available.

Saving a question immediately adds it to the queue. A pending message can be
edited or deleted before the agent retrieves it, and a thread can be closed,
reopened, or deleted in full after confirmation. Deletion permanently removes
the thread's messages and its unprocessed queue entries. Copy prompt
immediately copies the request for the agent and does not change the queue. On
wide screens, the panel slides over the content from the right and dims the
rest of the page; on phones, it fills the viewport. The panel closes on `Esc`,
through its close button, or when the backdrop is pressed, and returns focus to
the Discussions button.

Static builds and translation portals do not create the discussion menu.
Ordinary browser selection and copying remain available.

The risks page (`risks.md`) includes an additional summary of how many risks
remain open and which mitigations are complete. A risk's own state remains
separate from the checkboxes for individual mitigations.

On the roadmap page (`roadmap.md`), the main portal in `serve` mode shows a
button for adding one new `DLV-*` to an existing stage. It opens a dialog in the
current page. Static and translation portals do not include it.

## Transitions

<!-- toudocu:table transitions columns=id,useCase,action,condition,target,kind -->
| ID | Scenario | Action | Condition | Result | Type |
|---|---|---|---|---|---|
| TR-SITE-007 | UC-AGENT-FEEDBACK-01 | Open Changes | The user selected the shared Changes section | SC-CHANGES-WORKSPACE | navigation |

## Related documents

- [MOD-AGENT-FEEDBACK](../modules/MOD-AGENT-FEEDBACK.md)
- [UC-AGENT-FEEDBACK-01](../use-cases/UC-AGENT-FEEDBACK-01.md)
