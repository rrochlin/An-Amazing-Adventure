import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { pollWorldStatus } from "@/components/WaitForWorld";

// Mock WorldReady so tests run instantly without real HTTP or timers
vi.mock("@/services/api.game", () => ({
  WorldReady: vi.fn(),
}));

import { WorldReady } from "@/services/api.game";
const mockWorldReady = WorldReady as ReturnType<typeof vi.fn>;

// Suppress actual sleep delays
vi.mock("@/components/WaitForWorld", async (importOriginal) => {
  // Re-export with sleep mocked to 0
  const mod = await importOriginal<typeof import("@/components/WaitForWorld")>();
  return mod;
});

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
});

afterEach(() => vi.useRealTimers());

describe("pollWorldStatus", () => {
  it("returns true immediately when world is ready on first check", async () => {
    mockWorldReady.mockResolvedValue({ ready: true });
    const promise = pollWorldStatus("session-1");
    // Flush all timers/microtasks
    await vi.runAllTimersAsync();
    const result = await promise;
    expect(result).toBe(true);
    expect(mockWorldReady).toHaveBeenCalledWith("session-1");
  });

  it("retries until ready", async () => {
    mockWorldReady
      .mockResolvedValueOnce({ ready: false })
      .mockResolvedValueOnce({ ready: false })
      .mockResolvedValue({ ready: true });

    const promise = pollWorldStatus("session-2");
    await vi.runAllTimersAsync();
    const result = await promise;
    expect(result).toBe(true);
    expect(mockWorldReady).toHaveBeenCalledTimes(3);
  });

  it("returns false after max wait time exhausted", async () => {
    mockWorldReady.mockResolvedValue({ ready: false });
    const promise = pollWorldStatus("session-3");
    // Advance well past the 3 minute max
    await vi.advanceTimersByTimeAsync(300_000);
    const result = await promise;
    expect(result).toBe(false);
  });

  it("passes the session uuid to WorldReady", async () => {
    mockWorldReady.mockResolvedValue({ ready: true });
    const promise = pollWorldStatus("my-session-id");
    await vi.runAllTimersAsync();
    await promise;
    expect(mockWorldReady).toHaveBeenCalledWith("my-session-id");
  });
});
