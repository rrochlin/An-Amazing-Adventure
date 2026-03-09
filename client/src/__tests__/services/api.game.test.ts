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
  it("calls POST api/games with D&D creation params and returns session data", async () => {
    mockPost.mockResolvedValueOnce({
      data: { session_id: "sess-1", ready: false },
      status: 201,
    });
    const params = {
      name: "Legolas",
      race_id: "elf",
      subrace_id: "high-elf",
      class_id: "fighter",
      ability_scores: { str: 15, dex: 14, con: 13, int: 12, wis: 10, cha: 8 },
      selected_skills: ["athletics", "perception"],
    };
    const result = await CreateGame(params);
    expect(mockPost).toHaveBeenCalledWith("api/games", params);
    expect(result.session_id).toBe("sess-1");
    expect(result.ready).toBe(false);
  });

  it("calls POST api/games with optional world preferences", async () => {
    mockPost.mockResolvedValueOnce({
      data: { session_id: "sess-2", ready: false },
      status: 201,
    });
    const params = {
      name: "Aria",
      race_id: "half-orc",
      class_id: "barbarian",
      ability_scores: { str: 15, dex: 14, con: 13, int: 10, wis: 12, cha: 8 },
      selected_skills: ["athletics", "intimidation"],
      theme_hint: "gritty noir",
      preferences: ["stealth", "mystery"],
    };
    const result = await CreateGame(params);
    expect(mockPost).toHaveBeenCalledWith("api/games", params);
    expect(result.session_id).toBe("sess-2");
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

  it("returns optional metadata fields when present", async () => {
    const mockState = {
      current_room: { id: "r1", name: "Tavern", description: "", connections: {}, coordinates: { x: 0, y: 0, z: 0 }, items: [], occupants: [] },
      player: { id: "p1", name: "Hero", description: "", alive: true, health: 100, friendly: true, inventory: [] },
      rooms: {},
      chat_history: [],
    };
    mockGet.mockResolvedValueOnce({
      data: {
        session_id: "sess-1",
        ready: true,
        state: mockState,
        title: "The Shattered Kingdom",
        theme: "High fantasy",
        quest_goal: "Defeat the dark lord",
        total_tokens: 5000,
        conversation_count: 10,
        creation_params: { name: "Hero", race_id: "human", class_id: "fighter", ability_scores: {}, selected_skills: [], preferences: ["combat"] },
      },
      status: 200,
    });
    const result = await LoadGame("sess-1");
    expect(result.title).toBe("The Shattered Kingdom");
    expect(result.total_tokens).toBe(5000);
    expect(result.creation_params?.class_id).toBe("fighter");
    expect(result.creation_params?.preferences).toEqual(["combat"]);
  });
});

describe("DeleteGame", () => {
  it("calls DELETE api/games/{id}", async () => {
    mockDelete.mockResolvedValueOnce({ data: null, status: 204 });
    await DeleteGame("sess-1");
    expect(mockDelete).toHaveBeenCalledWith("api/games/sess-1");
  });
});
