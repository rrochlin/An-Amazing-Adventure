// Core domain types — mirror the server-v2 game engine view models.

// Legacy creation params (v1/v2 records)
export interface AdventureCreationParams {
   player_description?: string;
   player_age?: string;
   player_backstory?: string;
   theme_hint?: string;
   preferences?: string[];
}

// D&D 5e character creation data (v3+)
export interface CharacterCreationData {
   name: string;
   backstory?: string; // optional 2-3 sentence character backstory
   race_id: string; // e.g. "dwarf"
   subrace_id?: string; // e.g. "hill-dwarf"
   class_id: string; // "barbarian" | "fighter" | "monk"
   ability_scores: Record<string, number>; // "str","dex","con","int","wis","cha" -> value
   selected_skills: string[];
   theme_hint?: string;
   preferences?: string[];
}

// D&D 5e mechanical stats returned in CharacterView
export interface DnDStatsView {
   class_id: string;
   race_id: string;
   level: number;
   max_hp: number;
   ac: number;
   speed: number;
   proficiency_bonus: number;
   abilities: Record<string, number>; // "str","dex","con","int","wis","cha" -> score
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
   slot?: 'head' | 'chest' | 'legs' | 'hands' | 'feet' | 'back';
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
   dnd?: DnDStatsView; // present for v3+ characters
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
   player: CharacterView; // backward compat — same as self
   self?: CharacterView; // calling user's own character (v2+)
   party?: CharacterView[]; // other party members (v2+)
   rooms: Record<string, RoomView>;
   chat_history: ChatMessage[];
}

export interface WorldEvent {
   type: string; // "damage","heal","death","revive","item_gained","item_lost","item_appeared","character_arrived","character_departed"
   message: string; // human-readable, player's perspective
}

export interface ChatMessage {
   type: 'player' | 'narrative';
   content: string;
   events?: WorldEvent[]; // non-empty on narrative messages when world events occurred this turn
   /** ISO timestamp when the message was committed (client-side). Added on receive; absent for messages loaded from chat_history. */
   timestamp?: string;
}

// WebSocket frame types sent from server to client
export type WsFrameType =
   | 'narrative_chunk'
   | 'narrative_end'
   | 'game_state_update'
   | 'state_delta'
   | 'error'
   | 'streaming_blocked'
   | 'world_gen_log'
   | 'world_gen_ready';

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
   player?: CharacterView; // backward compat — same as self
   self?: CharacterView; // calling user's own character
   party?: CharacterView[]; // updated party member views
   updated_rooms?: Record<string, RoomView>;
   events?: WorldEvent[]; // player-visible world events this turn
   // new_message removed — narrative arrives via streaming frames, not state_delta
}

// Invite / party types
export interface InviteInfo {
   code: string;
   game_title: string;
   party_current: number;
   party_max: number;
   expired: boolean;
}

export interface CreateInviteResponse {
   code: string;
   url: string;
   expires: number; // Unix ms
}

// Legacy — kept for backward compat during transition
export type ChatMessageType = ChatMessage;
export type GameState = GameStateView;
