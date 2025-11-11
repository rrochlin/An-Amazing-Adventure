# React Client Application Architecture Summary

## Project Overview
This is a modern React adventure game client built with TypeScript, featuring a text-based narrative game with world mapping and real-time chat interactions. The application uses TanStack Router for navigation and Material-UI (MUI) for styling.

**Key Technologies:**
- React 19.0.0
- TanStack Router v1.130.2 (File-based routing)
- TypeScript 5.7.2
- Vite 6.3.5 (Build tool)
- Material-UI (MUI) v7.3.2
- Axios v1.11.0 (HTTP client)
- React Konva v19.0.7 (Canvas visualization)
- Zod v4.1.5 (Type validation)

---

## Directory Structure

```
client/
├── src/
│   ├── routes/              # File-based routing (TanStack Router)
│   │   ├── __root.tsx       # Root layout with header
│   │   ├── index.tsx        # Home/Game list page
│   │   ├── game-{$sessionUUID}.tsx  # Main game page (dynamic route)
│   │   ├── login.tsx        # Login page
│   │   ├── signup.tsx       # Signup page
│   │   └── profile.tsx      # User profile (stub)
│   │
│   ├── components/          # Reusable React components
│   │   ├── Header.tsx       # App header with navigation
│   │   ├── AccountPanel.tsx # User menu (profile, logout, theme)
│   │   ├── Chat.tsx         # Chat interface with markdown
│   │   ├── GameInfo.tsx     # Game state display (inventory, items, NPCs)
│   │   ├── RoomMap.tsx      # Map visualization using Konva
│   │   ├── CustomLink.tsx   # Custom router link component
│   │   ├── calcPosition.ts  # Force-directed layout for room positions
│   │   └── WaitForWorld.ts  # Poll world generation status
│   │
│   ├── services/            # API and business logic
│   │   ├── api.service.ts   # Base HTTP client (GET, POST, PUT, DELETE)
│   │   ├── api.game.ts      # Game API endpoints
│   │   ├── api.users.ts     # User/auth API endpoints
│   │   └── auth.service.ts  # Authentication & token management
│   │
│   ├── types/               # TypeScript type definitions
│   │   ├── types.ts         # Core domain types (GameState, Character, etc.)
│   │   └── api.types.ts     # API request/response types
│   │
│   ├── theme/
│   │   └── theme.ts         # MUI theme configuration (light/dark)
│   │
│   ├── main.tsx             # React app entry point
│   ├── routeTree.gen.ts     # Auto-generated route tree (TanStack Router)
│   └── reportWebVitals.ts   # Performance monitoring
│
├── package.json
├── tsconfig.json
├── vite.config.ts
└── index.html
```

---

## 1. Routing Setup (TanStack Router)

### Configuration
**File:** `src/main.tsx` & `src/routeTree.gen.ts`

TanStack Router uses **file-based routing**, automatically generating the route tree from files in the `src/routes/` directory.

```typescript
// Router initialization with features:
const router = createRouter({
  routeTree,
  context: {},
  defaultPreload: 'intent',        // Preload routes on intent
  scrollRestoration: true,          // Restore scroll position
  defaultStructuralSharing: true,   // Share data structures
  defaultPreloadStaleTime: 0,       // Preload immediately
})
```

### Routes

| Route | Purpose | Protection |
|-------|---------|-----------|
| `/` | Home/Game list | Requires auth |
| `/game-{$sessionUUID}` | Main game view | Requires auth |
| `/login` | User login | Public |
| `/signup` | User registration | Public |
| `/profile` | User profile | Requires auth (stub) |

### Route Protection Pattern
Routes use `beforeLoad` to enforce authentication:

```typescript
beforeLoad: () => {
  if (!isAuthenticated()) {
    throw redirect({
      to: "/login",
      search: { redirect: location.href },
    });
  }
}
```

### Dynamic Routes
The game route uses **path parameters** with type safety:
```typescript
// Route definition
export const Route = createFileRoute("/game-{$sessionUUID}")({...})

// Usage
const { sessionUUID } = Route.useParams()
navigate({
  to: "/game-{$sessionUUID}",
  params: { sessionUUID: game.sessionId },
})
```

---

## 2. Component Organization

### Component Hierarchy

```
__root (Root Layout)
├── Header
│   ├── CustomLink (Home)
│   └── AccountPanel
│       ├── Profile link
│       ├── Theme selector (dark/light/system)
│       └── Login/Logout
│
└── Outlet (Page-specific content)
    ├── index.tsx (Game List)
    │   └── Game dialog creation form
    │
    ├── login.tsx
    │   └── Login form
    │
    ├── signup.tsx
    │   └── Signup form
    │
    └── game-{$sessionUUID}.tsx (Main Game)
        ├── RoomMap (Konva canvas)
        └── Chat
            ├── Message display (Markdown)
            └── Command input
        └── GameInfo
            ├── Current room description
            ├── Inventory list
            ├── Room items list
            └── Occupants list
```

### Key Components

**Header.tsx**
- Global navigation bar
- Account menu dropdown
- Home link

**AccountPanel.tsx**
- User menu with logout
- Theme switcher (dark/light/system)
- Profile navigation
- Authentication state detection

**Chat.tsx**
- Scrollable message history
- Markdown rendering for narrative
- Command input with auto-focus
- Loading state indicator
- Keyboard: Enter to send

**GameInfo.tsx**
- Displays current room state
- Inventory management
- Item/NPC interaction placeholders
- Dark theme styling

**RoomMap.tsx**
- Canvas visualization using React Konva
- Force-directed layout for room positions
- Interactive hover tooltips
- Shows current location and connections

---

## 3. API Client Setup (Axios)

### Base HTTP Client
**File:** `src/services/api.service.ts`

Generic HTTP methods with automatic auth headers:

```typescript
export async function GET<T>(uri: string): Promise<AxiosResponse<T>>
export async function POST<T>(uri: string, body?: any): Promise<AxiosResponse<T>>
export async function PUT<T>(uri: string, body: any): Promise<AxiosResponse<T>>
export async function DELETE<T>(uri: string): Promise<AxiosResponse<T>>
```

**Features:**
- Automatic Bearer token injection
- Base URL from environment: `VITE_APP_URI`
- Generic type support for responses
- Error logging

### Game API
**File:** `src/services/api.game.ts`

```typescript
StartGame(body: ApiStartGameRequest): Promise<StartGameResponse>
  POST /games/{sessionUUID}

ListGames(): Promise<ListGamesResponse[]>
  GET /games

DescribeGame(sessionUUID: string): Promise<DescribeResponse>
  GET /describe/{sessionUUID}

DeleteGame(sessionUUID: string): Promise<boolean>
  DELETE /games/{sessionUUID}

Chat(sessionUUID: string, reqBody: ApiChatRequest): Promise<ChatResponse>
  POST /chat/{sessionUUID}

WorldReady(sessionUUID: string): Promise<WorldReadyResponse>
  GET /worldready/{sessionUUID}
```

### User/Auth API
**File:** `src/services/api.users.ts`

```typescript
Login(body: ApiLoginRequest): Promise<LoginResponse>
  POST /login

CreateNewUser(body: ApiCreateUserRequest): Promise<CreateUserResponse>
  POST /users

UpdateUser(body: ApiUpdateUserRequest): Promise<UpdateUserResponse>
  PUT /users
```

### Authentication Service
**File:** `src/services/auth.service.ts`

**Token Management:**
- Stores JWT + refresh token in localStorage (key: `AAA_JWT`)
- Token expiration tracking
- Automatic token refresh (30-min buffer before expiry)

**Key Functions:**
```typescript
getAuthHeaders(): AxiosRequestHeaders  // Get bearer token header
isAuthenticated(): boolean              // Check if user is logged in
ClearUserAuth(): void                   // Clear tokens on logout
refreshToken(): Promise<AxiosResponse>  // Refresh expired tokens
getJWT(): stored_tokens | undefined     // Get stored token object
```

**Token Object Structure:**
```typescript
interface stored_tokens {
  jwt: string              // Access token (1 hour lifetime)
  rtoken: string           // Refresh token
  expiresAt: number        // Expiration timestamp
}
```

---

## 4. State Management Patterns

### Local Component State
Uses React hooks for component-level state:

```typescript
const [gameState, setGameState] = useState<GameState | null>(null)
const [command, setCommand] = useState("")
const [chatHistory, setChatHistory] = useState<ChatMessageType[]>([])
const [isLoading, setIsLoading] = useState(false)
```

### Loader Data Pattern (TanStack Router)
Routes can fetch data via loaders and access with hooks:

```typescript
// Route definition
loader: async () => {
  const games = await ListGames()
  return games
}

// Component usage
const games = Route.useLoaderData()
```

### LocalStorage Persistence
Game state is cached locally:

```typescript
// Save game state
localStorage.setItem(`gameState-${sessionUUID}`, JSON.stringify(gameState))

// Load cached state
const localState = localStorage.getItem(`gameState-${sessionUUID}`)
if (localState) {
  state = JSON.parse(localState)
}
```

**Features:**
- Fast load on revisit
- Synced with server on fetch
- Fallback to server if missing

### No Global State Management
Application deliberately avoids Redux/Context for:
- Simpler architecture
- Per-session game state isolation
- Minimal provider overhead

---

## 5. Key Pages and Features

### Index Page (`/`)
**Purpose:** Game list and launcher

**Features:**
- Lists all active games
- Dialog to create new game (character name)
- Click to navigate to game
- Protected by authentication

**State Flow:**
1. Load games list via router loader
2. User clicks "Create Game" → Opens dialog
3. Submit form → `StartGame()` API call
4. New game added to list
5. Click game → Navigate to `/game-{sessionUUID}`

### Game Page (`/game-{$sessionUUID}`)
**Purpose:** Main gameplay interface

**Layout:**
- Left sidebar: Room map + game info
- Right panel: Chat interface
- Error alerts for connectivity issues

**State Management:**
- Initial game state from API
- Chat history from localStorage + API
- Polling for world generation status

**Workflows:**

1. **Load Game State:**
   - Poll `WorldReady()` until world generated
   - Fetch from localStorage if available
   - Request latest from `DescribeGame()` API
   - Merge with chat history

2. **Player Command:**
   - Add command to chat history
   - Send via `Chat()` API
   - Update gameState with response
   - Clear input field

3. **Room Visualization:**
   - Calculate positions with force-directed layout
   - Render rooms on Konva canvas
   - Show current room highlight
   - Display tooltips on hover

### Login Page (`/login`)
**Purpose:** User authentication

**Features:**
- Email/password form
- Form validation
- Success → stores tokens → redirects home
- Link to signup

### Signup Page (`/signup`)
**Purpose:** Account creation

**Features:**
- Email/password fields
- Password confirmation
- Minimum 6-char password validation
- Success → stores tokens → redirects to login
- Link to login

### Header & Account Panel
**Features:**
- Navigation to home
- Account menu dropdown
- Theme selector (dark/light/system)
- Logout functionality
- Conditional login/logout display

---

## 6. Type System

### Core Types (`types.ts`)

```typescript
interface GameState {
  current_room: Area
  player: Character
  visible_items?: { [key: string]: Item }
  visible_npcs?: { [key: string]: Character }
  connected_rooms?: string[]
  rooms?: { [key: string]: Area }
  chat_history?: ChatMessageType[]
}

interface Character {
  location: Area
  name: string
  description: string
  alive: boolean
  health: number
  inventory: Item[]
  friendly: boolean
}

interface Area {
  id: string
  connections: string[]
  items: Item[]
  occupants: string[]
  description: string
}

interface Item {
  name: string
  description: string
  weight: number
  location: Area | Character
}

interface ChatMessageType {
  type: "player" | "narrative"
  content: string
}

interface stored_tokens {
  jwt: string
  rtoken: string
  expiresAt: number
}
```

### API Types (`api.types.ts`)
- Request/Response pairs for each endpoint
- Type-safe API contracts
- Mirrors backend API structure
- Extends core types where applicable

---

## 7. Build & Development

### Vite Configuration
**File:** `vite.config.ts`

```typescript
plugins: [
  TanStackRouterVite({ autoCodeSplitting: true }),  // Auto route code-split
  viteReact()
]

resolve: {
  alias: {
    "@": resolve(__dirname, "./src")  // @ path alias
  }
}
```

### Scripts
```bash
npm run dev     # Development server (port 5173)
npm run build   # Production build + TypeScript check
npm run serve   # Preview production build
npm run test    # Run Vitest tests
```

### Environment Variables
```
VITE_APP_URI    # Backend API URL (default: http://localhost:8080)
```

### TypeScript Configuration
- Target: ES2022
- Strict mode enabled
- Path aliases (@/)
- Module resolution: bundler

---

## 8. Theme System

### MUI Theme Configuration
**File:** `src/theme/theme.ts`

**Features:**
- Light and dark color schemes
- CSS variables support
- System preference detection
- Customizable palette
- Border radius: 8px

**Available Modes:**
- Light (blue primary, white background)
- Dark (light blue primary, dark background)
- System (follows OS preference)

**Provider:** ThemeProvider wraps app in `__root.tsx`

---

## 9. Data Flow Overview

### Authentication Flow
```
1. User enters credentials → Login page
2. POST /login → Server validates
3. Response includes: jwt, refresh_token, expiry
4. Store in localStorage (AAA_JWT)
5. Subsequent requests include Bearer token
6. On 30-min expiry: POST /refresh auto-refreshes
7. Logout: Clear localStorage → Redirect to login
```

### Game Session Flow
```
1. Home page: GET /games → List user's sessions
2. Create game: POST /games/{uuid} with playerName
3. Poll GET /worldready/{uuid} until ready
4. Load game: GET /describe/{uuid} → Full game state
5. During play: POST /chat/{uuid} → Send command
6. Response includes updated game state
7. Cache in localStorage for offline access
```

### Component Rendering Flow
```
TanStack Router
  ↓
Load route data (loader function)
  ↓
Render component
  ↓
useEffect hooks fetch additional data
  ↓
Update local state
  ↓
Re-render with new state
  ↓
Cache to localStorage
```

---

## 10. Key Design Patterns

### Loader Pattern (Data Fetching)
Routes fetch data before rendering, preventing "no data" flash:
```typescript
loader: async () => ListGames()  // Fetch before component renders
```

### Polling Pattern
World generation status checked with exponential backoff:
```typescript
const pollWorldStatus = (sessionId) => {
  // Poll /worldready until ready
}
```

### Controlled Components
Forms use React state for input management:
```typescript
const [email, setEmail] = useState("")
<TextField value={email} onChange={(e) => setEmail(e.target.value)} />
```

### Error Boundaries
API errors displayed as inline alerts and logged to console:
```typescript
try {
  // API call
} catch (err) {
  setError("Failed to start game...")
  console.error(err)
}
```

### Type-Safe Navigation
TanStack Router provides full type safety:
```typescript
navigate({
  to: "/game-{$sessionUUID}",
  params: { sessionUUID: id }  // Type-checked params
})
```

---

## 11. Dependencies Overview

| Package | Version | Purpose |
|---------|---------|---------|
| react | ^19.0.0 | UI framework |
| @tanstack/react-router | ^1.130.2 | File-based routing |
| @mui/material | ^7.3.2 | Component library |
| axios | ^1.11.0 | HTTP client |
| react-konva | ^19.0.7 | Canvas visualization |
| zod | ^4.1.5 | Type validation |
| react-markdown | ^10.1.0 | Markdown rendering |
| typescript | ^5.7.2 | Type checking |
| vite | ^6.3.5 | Build tool |

---

## 12. Security Considerations

1. **Token Storage:** JWT stored in localStorage (consider HttpOnly cookies for production)
2. **Token Refresh:** Automatic refresh before expiry with 30-min buffer
3. **Authorization:** Bearer token in all API requests
4. **Route Protection:** Authentication checks in `beforeLoad` hooks
5. **CORS:** Assumes backend handles CORS for cross-origin requests

---

## Summary

This React application follows modern best practices with:
- **File-based routing** via TanStack Router for minimal boilerplate
- **Type-safe components** with TypeScript throughout
- **Modular services** for API communication
- **Local state management** without external state libraries
- **MUI for consistent styling** with light/dark theme support
- **Component-driven architecture** with clear separation of concerns
- **Data persistence** via localStorage caching
- **Protected routes** with authentication guards

The architecture is clean, scalable, and maintainable with clear boundaries between routing, components, services, and types.
