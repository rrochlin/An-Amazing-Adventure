// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWorldGenSocket } from "@/hooks/useWorldGenSocket";

// ── Mock auth ─────────────────────────────────────────────────────────────
vi.mock("@/services/auth.service", () => ({
  getStoredTokens: () => ({ accessToken: "test-token" }),
}));

// ── Minimal WebSocket mock ────────────────────────────────────────────────
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onmessage: ((e: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 1;
  closed = false;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    // Fire onopen on next microtask so useEffect can set the ref first
    Promise.resolve().then(() => this.onopen?.());
  }

  onopen: (() => void) | null = null;
  close() { this.closed = true; }

  receive(data: object) {
    act(() => {
      this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
    });
  }
}

beforeEach(() => {
  MockWebSocket.instances = [];
  vi.stubGlobal("WebSocket", MockWebSocket);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

// Flush microtasks so useEffect runs
const flushPromises = () => act(() => Promise.resolve());

describe("useWorldGenSocket", () => {
  it("does not open a socket when sessionId is null", async () => {
    renderHook(() => useWorldGenSocket({ sessionId: null, onReady: vi.fn() }));
    await flushPromises();
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("opens a socket with sessionId and token in URL", async () => {
    renderHook(() =>
      useWorldGenSocket({ sessionId: "sess-123", onReady: vi.fn() })
    );
    await flushPromises();
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toContain("sess-123");
    expect(MockWebSocket.instances[0].url).toContain("test-token");
  });

  it("appends world_gen_log frames to lines", async () => {
    const { result } = renderHook(() =>
      useWorldGenSocket({ sessionId: "sess-log", onReady: vi.fn() })
    );
    await flushPromises();
    const ws = MockWebSocket.instances[0];

    ws.receive({ type: "world_gen_log", payload: { line: "Summoning Architect..." } });
    ws.receive({ type: "world_gen_log", payload: { line: "Blueprint ready" } });

    expect(result.current.lines).toHaveLength(2);
    expect(result.current.lines[0]).toBe("Summoning Architect...");
    expect(result.current.lines[1]).toBe("Blueprint ready");
  });

  it("sets ready=true on world_gen_ready", async () => {
    const { result } = renderHook(() =>
      useWorldGenSocket({ sessionId: "sess-ready", onReady: vi.fn() })
    );
    await flushPromises();
    const ws = MockWebSocket.instances[0];

    expect(result.current.ready).toBe(false);
    ws.receive({ type: "world_gen_ready" });
    expect(result.current.ready).toBe(true);
  });

  it("calls onReady callback 800ms after world_gen_ready", async () => {
    vi.useFakeTimers();
    const onReady = vi.fn();
    renderHook(() => useWorldGenSocket({ sessionId: "sess-cb", onReady }));
    await act(() => Promise.resolve());

    const ws = MockWebSocket.instances[0];
    ws.receive({ type: "world_gen_ready" });

    expect(onReady).not.toHaveBeenCalled();
    act(() => vi.advanceTimersByTime(900));
    expect(onReady).toHaveBeenCalledTimes(1);
  });

  it("closes socket on unmount", async () => {
    const { unmount } = renderHook(() =>
      useWorldGenSocket({ sessionId: "sess-unmount", onReady: vi.fn() })
    );
    await flushPromises();
    const ws = MockWebSocket.instances[0];
    expect(ws.closed).toBe(false);
    unmount();
    expect(ws.closed).toBe(true);
  });

  it("appends error line on socket error", async () => {
    const { result } = renderHook(() =>
      useWorldGenSocket({ sessionId: "sess-err", onReady: vi.fn() })
    );
    await flushPromises();
    const ws = MockWebSocket.instances[0];

    act(() => ws.onerror?.());
    expect(result.current.lines[0]).toMatch(/ERROR/);
  });
});
