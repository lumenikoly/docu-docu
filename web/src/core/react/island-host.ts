export type IslandMount = (element: HTMLElement, context: {
  instance: string;
  signal: AbortSignal;
  model: unknown;
  onCleanup: (cleanup: () => void) => void;
}) => void | (() => void) | Promise<void | (() => void)>;

export type IslandRegistry = Record<string, () => Promise<{ mount: IslandMount }>>;

type ActiveIsland = {
  element: HTMLElement;
  controller: AbortController;
  cleanup?: () => void;
};

export class IslandHost {
  private readonly active = new Map<string, ActiveIsland>();
  private readonly discovered = new Map<string, HTMLElement>();
  private readonly failed = new WeakSet<HTMLElement>();

  constructor(private readonly registry: IslandRegistry) {}

  discover(scope: ParentNode = document): void {
    for (const element of scope.querySelectorAll<HTMLElement>("[data-td-island][data-td-island-instance]")) {
      const instance = element.dataset.tdIslandInstance!;
      if (!this.discovered.has(instance)) this.discovered.set(instance, element);
      if (element.dataset.tdIslandActivation !== "activated") void this.mount(element);
    }
  }

  activate(instance: string): Promise<void> {
    const element = this.discovered.get(instance);
    return element ? this.mount(element) : Promise.reject(new Error(`unknown island instance ${instance}`));
  }

  async mount(element: HTMLElement): Promise<void> {
    const name = element.dataset.tdIsland;
    const instance = element.dataset.tdIslandInstance;
    if (!name || !instance || this.active.has(instance) || (this.failed.has(element) && element.dataset.tdIslandActivation !== "activated")) return;
    const load = this.registry[name];
    if (!load) return;

    const controller = new AbortController();
    const active: ActiveIsland = { element, controller };
    this.active.set(instance, active);
    element.querySelector<HTMLElement>("[data-td-island-error]")?.setAttribute("hidden", "");
    element.dataset.tdIslandState = "loading";
    try {
      const module = await load();
      if (controller.signal.aborted || this.active.get(instance) !== active) return;
      const model = this.readModel(element);
      const cleanup = await module.mount(element, {
        instance,
        signal: controller.signal,
        model,
        onCleanup: (value) => { active.cleanup = value; },
      });
      if (controller.signal.aborted || this.active.get(instance) !== active) {
        if (typeof cleanup === "function") cleanup();
        return;
      }
      if (typeof cleanup === "function") active.cleanup = cleanup;
      element.dataset.tdIslandState = "mounted";
    } catch {
      if (this.active.get(instance) === active) {
        this.unmount(instance);
        if (element.dataset.tdIslandActivation !== "activated") this.failed.add(element);
        element.dataset.tdIslandState = "error";
        element.querySelector<HTMLElement>("[data-td-island-error]")?.removeAttribute("hidden");
      }
    }
  }

  unmount(instance: string): void {
    const active = this.active.get(instance);
    if (!active) return;
    this.active.delete(instance);
    active.controller.abort();
    try { active.cleanup?.(); } catch { /* one feature cannot block the remaining cleanup */ }
    delete active.element.dataset.tdIslandState;
  }

  unmountAll(): void {
    for (const instance of [...this.active.keys()]) this.unmount(instance);
    this.discovered.clear();
  }

  private readModel(element: HTMLElement): unknown {
    const id = element.dataset.tdIslandModel;
    if (!id) return undefined;
    const source = document.getElementById(id);
    if (!(source instanceof HTMLScriptElement) || source.type !== "application/json") throw new Error("invalid island model");
    return JSON.parse(source.textContent || "null");
  }
}

const registry: IslandRegistry = {};
export const islandHost = new IslandHost(registry);
export function registerIsland(name: string, load: IslandRegistry[string]): void { registry[name] = load; }
