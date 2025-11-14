export interface GameState {
  current_room: Area;
  player: Character;
  visible_items?: { [key: string]: Item };
  visible_npcs?: { [key: string]: Character };
  connected_rooms?: string[];
  rooms?: { [key: string]: Area };
  chat_history?: ChatMessageType[];
  map_images?: { [key: string]: string }; // map of image type -> S3 URL
}

export interface Character {
  location: Area;
  name: string;
  description: string;
  alive: boolean;
  health: number;
  inventory: Item[];
  friendly: boolean;
}

export interface Coordinates {
  x: number;
  y: number;
  z: number;
}

export interface Area {
  id: string;
  connections: { [direction: string]: string }; // direction -> room_id
  coordinates: Coordinates;
  items: Item[];
  occupants: string[];
  description: string;
}

export interface Item {
  name: string;
  description: string;
  weight: number;
  // can't do this in Go but it's supposed to only be these 2
  location: Area | Character;
}

export interface ChatMessageType {
  type: "player" | "narrative";
  content: string;
}

export interface stored_tokens {
  jwt: string;
  rtoken: string;
  expiresAt: number;
}
