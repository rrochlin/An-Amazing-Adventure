import { DELETE, GET, POST } from "./api.service";
import type { GameStateView, AdventureCreationParams } from "../types/types";

export interface GameListItem {
  session_id: string;
  player_name: string;
  ready: boolean;
  title?: string;
  theme?: string;
  quest_goal?: string;
  conversation_count?: number;
  total_tokens?: number;
}

export interface GameLoadResponse {
  session_id: string;
  ready: boolean;
  state: GameStateView;
  title?: string;
  theme?: string;
  quest_goal?: string;
  total_tokens?: number;
  conversation_count?: number;
  creation_params?: AdventureCreationParams;
}

export interface CreateGameParams {
  player_name?: string;
  player_description?: string;
  player_age?: string;
  player_backstory?: string;
  theme_hint?: string;
  preferences?: string[];
}

export async function ListGames(): Promise<GameListItem[]> {
  const res = await GET<GameListItem[]>("api/games");
  return res.data;
}

export async function CreateGame(
  params: CreateGameParams,
): Promise<{ session_id: string; ready: boolean }> {
  const res = await POST<{ session_id: string; ready: boolean }>("api/games", params);
  return res.data;
}

export async function LoadGame(sessionId: string): Promise<GameLoadResponse> {
  const res = await GET<GameLoadResponse>(`api/games/${sessionId}`);
  return res.data;
}

export async function DeleteGame(sessionId: string): Promise<void> {
  await DELETE(`api/games/${sessionId}`);
}
