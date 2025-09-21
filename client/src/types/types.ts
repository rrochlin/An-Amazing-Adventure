export interface GameState {
  current_room: Area;
  player: Character;
  visible_items?: { [key: string]: Item };
  visible_npcs?: { [key: string]: Character };
  connected_rooms?: string[];
  rooms?: { [key: string]: Area };
  chat_history?: ChatMessageType[];
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

export interface Area {
  id: string;
  connections: string[];
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
