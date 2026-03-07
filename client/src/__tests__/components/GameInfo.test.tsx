// @vitest-environment jsdom
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider } from "@mui/material/styles";
import { AppTheme } from "@/theme/theme";
import { GameInfo } from "@/components/GameInfo";
import type { GameStateView } from "@/types/types";

function makeGameState(overrides: Partial<GameStateView> = {}): GameStateView {
  return {
    current_room: {
      id: "room-1",
      name: "The Rusty Tavern",
      description: "A smoky room with a bar.",
      connections: { north: "room-2" },
      coordinates: { x: 0, y: 0, z: 0 },
      items: [
        { id: "item-1", name: "Rusty Dagger", description: "Dull blade", weight: 1, equippable: false },
      ],
      occupants: [
        { id: "npc-1", name: "Barkeep", description: "Tired man", alive: true, health: 100, friendly: true, inventory: [] },
      ],
    },
    player: {
      id: "p1",
      name: "Aragorn",
      description: "A ranger",
      alive: true,
      health: 80,
      friendly: true,
      inventory: [
        { id: "item-2", name: "Health Potion", description: "Restores health", weight: 0.5, equippable: false },
      ],
    },
    rooms: {},
    chat_history: [],
    ...overrides,
  };
}

function renderInfo(state: GameStateView | null, onItemClick = vi.fn()) {
  return render(
    <ThemeProvider theme={AppTheme}>
      <GameInfo gameState={state} onItemClick={onItemClick} />
    </ThemeProvider>
  );
}

describe("GameInfo", () => {
  it("renders null state without crashing", () => {
    renderInfo(null);
    // Should render tabs without throwing
    expect(screen.getByText(/room/i)).toBeInTheDocument();
  });

  it("shows current room name", () => {
    renderInfo(makeGameState());
    expect(screen.getByText("The Rusty Tavern")).toBeInTheDocument();
  });

  it("shows room items on Room tab", async () => {
    renderInfo(makeGameState());
    await userEvent.click(screen.getByRole("tab", { name: /room/i }));
    expect(screen.getByText("Rusty Dagger")).toBeInTheDocument();
  });

  it("shows room occupants on Room tab", async () => {
    renderInfo(makeGameState());
    await userEvent.click(screen.getByRole("tab", { name: /room/i }));
    expect(screen.getByText("Barkeep")).toBeInTheDocument();
  });

  it("shows player inventory on Inventory tab", async () => {
    renderInfo(makeGameState());
    // Find and click the Inventory tab
    const inventoryTab = screen.getByRole("tab", { name: /inventory/i });
    await userEvent.click(inventoryTab);
    expect(screen.getByText("Health Potion")).toBeInTheDocument();
  });

  it("shows empty inventory message when inventory is empty", async () => {
    const state = makeGameState();
    state.player.inventory = [];
    renderInfo(state);
    await userEvent.click(screen.getByRole("tab", { name: /inventory/i }));
    expect(screen.getByText(/your inventory is empty/i)).toBeInTheDocument();
  });

  it("shows empty room items message when no items", async () => {
    const state = makeGameState();
    state.current_room.items = [];
    renderInfo(state);
    await userEvent.click(screen.getByRole("tab", { name: /room/i }));
    expect(screen.getByText(/no items in this room/i)).toBeInTheDocument();
  });

  it("shows empty occupants message when room has no NPCs", async () => {
    const state = makeGameState();
    state.current_room.occupants = [];
    renderInfo(state);
    await userEvent.click(screen.getByRole("tab", { name: /room/i }));
    expect(screen.getByText(/no occupants in this room/i)).toBeInTheDocument();
  });
});
