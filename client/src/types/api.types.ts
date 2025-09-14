import type { GameState } from "./types";

// POST startgame
export interface ApiStartGameRequest {
  playerName: string;
}
export interface ApiStartGameResponse {
  error?: string;
  ready?: boolean;
}
export interface StartGameResponse {
  sessionUUID: string;
  success: boolean;
  error?: string;
  ready: boolean;
}

// GET games
export interface ListGamesResponse {
  sessionId: string;
  playerName: string;
}
export interface ApiListGamesResponse extends ListGamesResponse {}

// GET describe
export interface RoomInfo {
  id: string;
  description: string;
  connections: string[];
  items: string[];
  occupants: string[];
}

export interface ApiDescribeRequest {}
export interface ApiDescribeResponse {
  description: string;
  current_room: string;
  rooms: Record<string, RoomInfo>;
  game_state: GameState;
}
export interface DescribeResponse extends ApiDescribeResponse {}
// POST chat
export interface ApiChatRequest {
  chat: string;
}
export interface ApiChatResponse {
  Response: string;
  NewAreas?: Record<string, RoomInfo>;
  game_state: GameState;
}
export interface ChatResponse extends ApiChatResponse {}

// GET worldready
export interface ApiWorldReadyRequest {}
export interface ApiWorldReadyResponse {}
export interface WorldReadyResponse {
  ready: boolean;
}

// POST login
export interface ApiLoginRequest {
  email: string;
  password: string;
}

export interface ApiLoginResponse {
  id: string;
  created_at: string; // this is the user account creation
  updated_at: string; // this is the user account update
  email: string;
  token?: string; // tokens have a life of 1 hour
  refresh_token?: string;
}
export interface LoginResponse {
  success: boolean;
}
// POST /api/refresh
export interface ApiRefreshRequest {}
export interface ApiRefreshResponse {
  token: string;
}
export interface RefreshResponse extends ApiRefreshResponse {}
// POST /api/revoke
export interface ApiRevokeRequest {}
export interface ApiRevokeResponse {}
export interface RevokeResponse {
  success: boolean;
}
// PUT /api/users
export interface ApiUpdateUserRequest {}
export interface ApiUpdateUserResponse extends ApiLoginResponse {}
export interface UpdateUserResponse {
  success: boolean;
}
// POST /api/users
export interface ApiCreateUserRequest {
  email: string;
  password: string;
}
export interface ApiCreateUserResponse extends ApiLoginResponse {}
export interface CreateUserResponse {
  success: boolean;
}
