// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock the HTTP helpers so no real requests go out
vi.mock("@/services/api.service", () => ({
  GET: vi.fn(),
  POST: vi.fn(),
  PUT: vi.fn(),
  DELETE: vi.fn(),
}));

vi.mock("@/services/auth.service", () => ({
  getAuthHeader: () => "Bearer test-token",
  refreshSession: vi.fn(),
  ClearUserAuth: vi.fn(),
}));

import * as apiService from "@/services/api.service";
import {
  ListGames,
  CreateGame,
  LoadGame,
  DeleteGame,
  WorldReady,
} from "@/services/api.game";

const mockGet = apiService.GET as ReturnType<typeof vi.fn>;
const mockPost = apiService.POST as ReturnType<typeof vi.fn>;
const mockDelete = apiService.DELETE as ReturnType<typeof vi.fn>;

beforeEach(() => vi.clearAllMocks());

describe("ListGames", () => {
  it("calls GET api/games and returns data", async () => {
    const games = [{ session_id: "abc", player_name: "Hero", ready: true }];
    mockGet.mockResolvedValueOnce({ data: games, status: 200 });
    const result = await ListGames();
    expect(mockGet).toHaveBeenCalledWith("api/games");
    expect(result).toEqual(games);
  });

  it("propagates errors from GET", async () => {
    mockGet.mockRejectedValueOnce(new Error("network error"));
    await expect(ListGames()).rejects.toThrow("network error");
  });
});

describe("CreateGame", () => {
  it("calls POST api/games with player_name and returns session data", async () => {
    mockPost.mockResolvedValueOnce({
      data: { session_id: "sess-1", ready: false },
      status: 201,
    });
    const result = await CreateGame("Legolas");
    expect(mockPost).toHaveBeenCalledWith("api/games", { player_name: "Legolas" });
    expect(result.session_id).toBe("sess-1");
    expect(result.ready).toBe(false);
  });
});

describe("LoadGame", () => {
  it("calls GET api/games/{id} and returns response", async () => {
    const mockState = {
      current_room: { id: "r1", name: "Tavern", description: "", connections: {}, coordinates: { x: 0, y: 0, z: 0 }, items: [], occupants: [] },
      player: { id: "p1", name: "Hero", description: "", alive: true, health: 100, friendly: true, inventory: [] },
      rooms: {},
      chat_history: [],
    };
    mockGet.mockResolvedValueOnce({
      data: { session_id: "sess-1", ready: true, state: mockState },
      status: 200,
    });
    const result = await LoadGame("sess-1");
    expect(mockGet).toHaveBeenCalledWith("api/games/sess-1");
    expect(result.session_id).toBe("sess-1");
    expect(result.ready).toBe(true);
  });
});

describe("DeleteGame", () => {
  it("calls DELETE api/games/{id}", async () => {
    mockDelete.mockResolvedValueOnce({ data: null, status: 204 });
    await DeleteGame("sess-1");
    expect(mockDelete).toHaveBeenCalledWith("api/games/sess-1");
  });
});

describe("WorldReady", () => {
  it("returns ready:true on 200", async () => {
    mockGet.mockResolvedValueOnce({ status: 200 });
    const result = await WorldReady("sess-1");
    expect(result.ready).toBe(true);
  });

  it("returns ready:false on 204", async () => {
    mockGet.mockResolvedValueOnce({ status: 204 });
    const result = await WorldReady("sess-1");
    expect(result.ready).toBe(false);
  });

  it("returns ready:false on network error", async () => {
    mockGet.mockRejectedValueOnce(new Error("timeout"));
    const result = await WorldReady("sess-1");
    expect(result.ready).toBe(false);
  });
});
