import { afterEach, describe, expect, test } from "vitest";
import { IslandHost } from "../src/core/react/island-host";

afterEach(() => { document.body.replaceChildren(); });

describe("IslandHost", () => {
  test("discovers eager islands once and activates deferred islands explicitly", async () => {
    const mounted: string[] = [];
    const host = new IslandHost({ demo: async () => ({ mount: async (_element, context) => { mounted.push(context.instance); } }) });
    document.body.innerHTML = '<div data-td-island="demo" data-td-island-instance="eager"></div><div data-td-island="demo" data-td-island-instance="lazy" data-td-island-activation="activated"></div>';
    host.discover();
    host.discover();
    await Promise.resolve();
    expect(mounted).toEqual(["eager"]);
    await host.activate("lazy");
    expect(mounted).toEqual(["eager", "lazy"]);
  });

  test("unmounts every instance and prevents stale async mounts", async () => {
    const cleanups: string[] = [];
    let release!: () => void;
    const pending = new Promise<void>((resolve) => { release = resolve; });
    const host = new IslandHost({ demo: async () => ({ mount: async (_element, context) => { await pending; return () => cleanups.push(context.instance); } }) });
    document.body.innerHTML = '<div data-td-island="demo" data-td-island-instance="a"></div>';
    host.discover();
    await new Promise<void>((resolve) => queueMicrotask(resolve));
    host.unmountAll();
    release();
    await pending;
    await Promise.resolve();
    expect(cleanups).toEqual(["a"]);
    expect(document.querySelector("[data-td-island-state]")).toBeNull();
  });

  test("isolates invalid model failures and retries activated islands", async () => {
    const host = new IslandHost({ demo: async () => ({ mount: () => undefined }) });
    document.body.innerHTML = '<div data-td-island="demo" data-td-island-instance="a" data-td-island-activation="activated" data-td-island-model="bad"></div><script id="bad" type="application/json">{</script>';
    host.discover();
    await host.activate("a");
    expect(document.querySelector<HTMLElement>("[data-td-island-instance=a]")?.dataset.tdIslandState).toBe("error");
    document.getElementById("bad")!.textContent = "{}";
    await host.activate("a");
    expect(document.querySelector<HTMLElement>("[data-td-island-instance=a]")?.dataset.tdIslandState).toBe("mounted");
  });

  test("does not retry a failed eager island on the same page", async () => {
    let loads = 0;
    const host = new IslandHost({ demo: async () => { loads += 1; throw new Error("broken"); } });
    document.body.innerHTML = '<div data-td-island="demo" data-td-island-instance="a"></div>';
    host.discover();
    await new Promise<void>((resolve) => queueMicrotask(resolve));
    host.discover();
    await new Promise<void>((resolve) => queueMicrotask(resolve));
    expect(loads).toBe(1);
  });

  test("A to B cycles abort old effects without duplicate handlers", async () => {
    let events = 0;
    const signals: AbortSignal[] = [];
    const listener = () => { events += 1; };
    const host = new IslandHost({ demo: async () => ({ mount: (_element, context) => {
      signals.push(context.signal);
      document.addEventListener("demo", listener);
      return () => document.removeEventListener("demo", listener);
    } }) });

    for (const instance of ["a", "b", "a", "b"]) {
      document.body.innerHTML = `<div data-td-island="demo" data-td-island-instance="${instance}"></div>`;
      host.discover();
      await new Promise<void>((resolve) => queueMicrotask(resolve));
      document.dispatchEvent(new Event("demo"));
      host.unmountAll();
    }

    document.dispatchEvent(new Event("demo"));
    expect(events).toBe(4);
    expect(signals).toHaveLength(4);
    expect(signals.every((signal) => signal.aborted)).toBe(true);
  });

  test("continues unmounting after one cleanup throws", async () => {
    const cleaned: string[] = [];
    const host = new IslandHost({ demo: async () => ({ mount: (_element, context) => () => {
      cleaned.push(context.instance);
      if (context.instance === "a") throw new Error("broken cleanup");
    } }) });
    document.body.innerHTML = '<div data-td-island="demo" data-td-island-instance="a"></div><div data-td-island="demo" data-td-island-instance="b"></div>';
    host.discover();
    await new Promise<void>((resolve) => queueMicrotask(resolve));
    host.unmountAll();
    expect(cleaned).toEqual(["a", "b"]);
  });

  test("cleans an allocated root when first render throws and shows localized fallback", async () => {
    let roots = 0;
    const host = new IslandHost({ demo: async () => ({ mount: (_element, context) => {
      roots += 1;
      context.onCleanup(() => { roots -= 1; });
      throw new Error("render failed");
    } }) });
    document.body.innerHTML = '<div data-td-island="demo" data-td-island-instance="a" data-td-island-activation="activated"><p data-td-island-error hidden>Не удалось открыть.</p></div>';
    host.discover();
    await host.activate("a");
    expect(roots).toBe(0);
    expect(document.querySelector<HTMLElement>("[data-td-island-error]")?.hidden).toBe(false);
    await host.activate("a");
    expect(roots).toBe(0);
  });
});
