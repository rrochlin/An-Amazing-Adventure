// Core domain types — mirror the server-v2 game engine view models.

export interface Coordinates {
  x: number;
  y: number;
  z: number;
}

export interface ItemView {
  id: string;
  name: string;
  description: string;
  weight: number;
  equippable: boolean;
  slot?: "head" | "chest" | "legs" | "hands" | "feet" | "back";
}

export interface CharacterView {
  id: string;
  name: string;
  description: string;
  alive: boolean;
  health: number;
  friendly: boolean;
  inventory: ItemView[];
}

export interface RoomView {
  id: string;
  name: string;
  description: string;
  connections: Record<string, string>; // direction -> room ID
  coordinates: Coordinates;
  items: ItemView[];
  occupants: CharacterView[];
}

export interface GameStateView {
  current_room: RoomView;
  player: CharacterView;
  rooms: Record<string, RoomView>;
  chat_history: ChatMessage[];
}

export interface ChatMessage {
  type: "player" | "narrative";
  content: string;
}

// WebSocket frame types sent from server to client
export type WsFrameType =
  | "narrative_chunk"
  | "narrative_end"
  | "game_state_update"
  | "state_delta"
  | "error"
  | "streaming_blocked";

export interface WsFrame {
  type: WsFrameType;
  payload?: unknown;
}

export interface NarrativeChunkPayload {
  content: string;
}

export interface StateDelta {
  current_room?: RoomView;
  player?: CharacterView;
  updated_rooms?: Record<string, RoomView>;
  new_message?: ChatMessage;
}

// Legacy — kept for backward compat during transition
export type ChatMessageType = ChatMessage;
export type GameState = GameStateView;
