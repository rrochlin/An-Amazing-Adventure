export interface RoomInfo {
	id: string;
	description: string;
	connections: string[];
	items: string[];
	occupants: string[];
}

export interface GameState {
	description: string;
	inventory: string[];
	health: number;
	position: { x: number; y: number };
	current_room: string;
	rooms: { [key: string]: RoomInfo };
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
