import { DELETE, GET, POST } from './api.service';
import type { GameStateView, CharacterCreationData } from '../types/types';

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

export interface UserQuotaInfo {
   tokens_used: number;
   token_limit: number; // 0 = unlimited
   ai_enabled: boolean;
   role: string;
}

export interface ListGamesResponse {
   games: GameListItem[];
   user_quota: UserQuotaInfo;
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
   creation_params?: CharacterCreationData;
   needs_character_reset?: boolean;
}

export interface CreateGameResponse {
   session_id: string;
   ready: boolean;
   preview_mode: boolean;
}

export async function ListGames(): Promise<ListGamesResponse> {
   const res = await GET<ListGamesResponse>('api/games');
   return res.data;
}

export async function CreateGame(
   params: CharacterCreationData,
): Promise<CreateGameResponse> {
   const res = await POST<CreateGameResponse>('api/games', params);
   return res.data;
}

export async function LoadGame(sessionId: string): Promise<GameLoadResponse> {
   const res = await GET<GameLoadResponse>(`api/games/${sessionId}`);
   return res.data;
}

export async function DeleteGame(sessionId: string): Promise<void> {
   await DELETE(`api/games/${sessionId}`);
}

/** Re-trigger world generation for a stuck not-ready game. */
export async function RetryWorldGen(sessionId: string): Promise<void> {
   await POST(`api/games/${sessionId}/retry-world-gen`, {});
}

/** Update a joined party member's character with their D&D creation data. */
export async function JoinCharacter(
   sessionId: string,
   params: CharacterCreationData,
): Promise<{ session_id: string }> {
   const res = await POST<{ session_id: string }>(
      `api/games/${sessionId}/join-character`,
      params,
   );
   return res.data;
}
