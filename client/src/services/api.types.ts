// POST startgame
export interface ApiStartGameRequest {
  uuid: string;
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
}
export interface DescribeResponse extends ApiDescribeRequest {}
// POST chat
export interface ApiChatRequest {
  chat: string;
}
export interface ApiChatResponse {
  Response: string;
  NewAreas?: Record<string, RoomInfo>;
  game_state: {
    current_room: string;
    inventory: string[];
    rooms: Record<string, RoomInfo>;
  };
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
  username: string;
  password: string;
}

export interface ApiLoginResponse {
  id: string;
  created_at: Date;
  updated_at: Date;
  email: string;
  token?: string;
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
