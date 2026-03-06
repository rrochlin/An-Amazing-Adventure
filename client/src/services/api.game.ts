import { DELETE, GET, POST } from "./api.service";
import type { GameStateView } from "../types/types";

export interface GameListItem {
  session_id: string;
  player_name: string;
  ready: boolean;
}

export interface GameLoadResponse {
  session_id: string;
  ready: boolean;
  state: GameStateView;
}

export async function ListGames(): Promise<GameListItem[]> {
  const res = await GET<GameListItem[]>("api/games");
  return res.data;
}

export async function CreateGame(
  playerName: string,
): Promise<{ session_id: string; ready: boolean }> {
  const res = await POST<{ session_id: string; ready: boolean }>("api/games", {
    player_name: playerName,
  });
  return res.data;
}

export async function LoadGame(sessionId: string): Promise<GameLoadResponse> {
  const res = await GET<GameLoadResponse>(`api/games/${sessionId}`);
  return res.data;
}

export async function DeleteGame(sessionId: string): Promise<void> {
  await DELETE(`api/games/${sessionId}`);
}

export async function WorldReady(
  sessionId: string,
): Promise<{ ready: boolean }> {
  try {
    const res = await GET<void>(`api/worldready/${sessionId}`);
    return { ready: res.status === 200 };
  } catch {
    return { ready: false };
  }
}
