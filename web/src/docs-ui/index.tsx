import type { ReactNode } from "react";
import { Diagnostic, EmptyState } from "../ui";

export type Translator = (key: string, values?: readonly unknown[]) => string;
export type DiagnosticViewModel = { kind: "empty" | "error"; titleKey: string; detailKey?: string };

export function DiagnosticSummary({ model, translate }: { model: DiagnosticViewModel; translate: Translator }): ReactNode {
  const content = model.detailKey ? <p>{translate(model.detailKey)}</p> : null;
  const title = translate(model.titleKey);
  return model.kind === "error"
    ? <Diagnostic title={title}>{content}</Diagnostic>
    : <EmptyState title={title}>{content}</EmptyState>;
}
