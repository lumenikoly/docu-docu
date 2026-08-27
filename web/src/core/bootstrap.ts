import { catalog } from "./locale";

export const PAGE_KINDS = [
  "document",
  "architecture",
  "module",
  "use-case",
  "flow",
  "screen",
  "standard",
  "runbook",
  "task",
] as const;

export type PageKind = (typeof PAGE_KINDS)[number];
export type Runtime = "static" | "serve";

export interface PageBootstrap {
  schemaVersion: 1;
  runtime: Runtime;
  page: { kind: PageKind; id?: string; path: string };
  portal: { assetBase: string; dataBase: string };
  ui: {
    locale: string;
    theme: string;
    colorScheme: string;
    accent: string;
    density: string;
    contentWidth: string;
  };
  capabilities: {
    search: boolean;
    diagrams: boolean;
    editor: boolean;
    changes: boolean;
    review: boolean;
    rebuild: boolean;
    taskWorkspace: boolean;
    updateCheck: boolean;
    agentConsole: boolean;
    taskActions: boolean;
  };
  endpoints?: Partial<Record<"editor" | "changes" | "review" | "rebuild" | "version" | "agentConsole" | "taskActions", string>>;
}

export type BootstrapResult =
  | { ok: true; value: PageBootstrap }
  | { ok: false; reason: "bootstrap unavailable" | "unsupported schema" | "invalid bootstrap" };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function parseBootstrap(value: unknown): BootstrapResult {
  if (!isRecord(value)) return { ok: false, reason: "invalid bootstrap" };
  if (value.schemaVersion !== 1) return { ok: false, reason: "unsupported schema" };
  if (value.runtime !== "static" && value.runtime !== "serve") return { ok: false, reason: "invalid bootstrap" };
  if (!isRecord(value.page) || !PAGE_KINDS.includes(value.page.kind as PageKind) || typeof value.page.path !== "string") {
    return { ok: false, reason: "invalid bootstrap" };
  }
  if (!isRecord(value.portal) || typeof value.portal.assetBase !== "string" || typeof value.portal.dataBase !== "string") {
    return { ok: false, reason: "invalid bootstrap" };
  }
  if (!isRecord(value.ui) || typeof value.ui.locale !== "string" || !isRecord(value.capabilities)) {
    return { ok: false, reason: "invalid bootstrap" };
  }
  return { ok: true, value: value as unknown as PageBootstrap };
}

export function readBootstrap(root: Document = document): BootstrapResult {
  const node = root.getElementById("toudocu-page");
  if (!node?.textContent) return { ok: false, reason: "bootstrap unavailable" };
  try {
    return parseBootstrap(JSON.parse(node.textContent));
  } catch {
    return { ok: false, reason: "invalid bootstrap" };
  }
}

const result = readBootstrap();
if (result.ok) window.ToudocuPage = result.value;
else {
  document.documentElement.dataset.bootstrapError = result.reason;
  const messages = catalog(document.documentElement.lang);
  const status = document.createElement("p");
  status.className = "bootstrap-error";
  status.dataset.uiState = "error";
  status.setAttribute("role", "status");
  status.textContent = result.reason === "unsupported schema" ? messages.unsupportedSchema : messages.bootstrapUnavailable;
  document.body.prepend(status);
}
