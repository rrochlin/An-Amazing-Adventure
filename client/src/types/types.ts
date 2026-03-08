// Core domain types — mirror the server-v2 game engine view models.

export interface AdventureCreationParams {
  player_description?: string;
  player_age?: string;
  player_backstory?: string;
  theme_hint?: string;
  preferences?: string[];
}

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

export interface EquipmentView {
  head?: ItemView;
  chest?: ItemView;
  legs?: ItemView;
  hands?: ItemView;
  feet?: ItemView;
  back?: ItemView;
}

export interface CharacterView {
  id: string;
  name: string;
  description: string;
  alive: boolean;
  health: number;
  friendly: boolean;
  inventory: ItemView[];
  equipment: EquipmentView;
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

export interface WorldEvent {
  type: string; // "damage","heal","death","revive","item_gained","item_lost","item_appeared","character_arrived","character_departed"
  message: string; // human-readable, player's perspective
}

export interface ChatMessage {
  type: "player" | "narrative";
  content: string;
  events?: WorldEvent[]; // non-empty on narrative messages when world events occurred this turn
  /** ISO timestamp when the message was committed (client-side). Added on receive; absent for messages loaded from chat_history. */
  timestamp?: string;
}

// WebSocket frame types sent from server to client
export type WsFrameType =
  | "narrative_chunk"
  | "narrative_end"
  | "game_state_update"
  | "state_delta"
  | "error"
  | "streaming_blocked"
  | "world_gen_log"
  | "world_gen_ready";

export interface WsFrame {
  type: WsFrameType;
  payload?: unknown;
}

export interface NarrativeChunkPayload {
  content: string;
}

export interface WorldGenLogPayload {
  line: string;
}

export interface StateDelta {
  current_room?: RoomView;
  player?: CharacterView;
  updated_rooms?: Record<string, RoomView>;
  events?: WorldEvent[]; // player-visible world events this turn
  // new_message removed — narrative arrives via streaming frames, not state_delta
}

// Legacy — kept for backward compat during transition
export type ChatMessageType = ChatMessage;
export type GameState = GameStateView;
