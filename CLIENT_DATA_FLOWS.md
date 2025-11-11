# React Client Application - Data Flow Diagrams

## 1. Authentication Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION LIFECYCLE                      │
└─────────────────────────────────────────────────────────────────┘

LOGIN PROCESS:
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│  Login Page  │─POST /─→│   Backend    │─────→  │  Token JSON  │
│              │  login  │     API      │        │              │
└──────────────┘         └──────────────┘        └──────────────┘
                                                         │
                                                         ↓
                                              ┌──────────────────┐
                                              │  localStorage    │
                                              │  AAA_JWT: {      │
                                              │    jwt: token,   │
                                              │    rtoken: ...,  │
                                              │    expiresAt: .. │
                                              │  }               │
                                              └──────────────────┘

TOKEN REFRESH (Automatic):
┌─────────────────────────────────────────┐
│  API Request in api.service.ts          │
│  getAuthHeaders() called                │
└─────────────────────────────────────────┘
                    ↓
        ┌───────────────────────┐
        │ Check token expiry    │
        │ expiresAt < Date.now()│
        └───────────────────────┘
                    ↓
        ┌──────────────────────────────────┐
        │ Within 30 min of expiry?         │
        │ expiresAt + 30*60*1000 < Now()   │
        └──────────────────────────────────┘
                    ↓
        ┌──────────────────────────────────┐
        │ POST /refresh with rtoken        │
        │ Response: new jwt token          │
        └──────────────────────────────────┘
                    ↓
        ┌──────────────────────────────────┐
        │ Update localStorage with new jwt │
        └──────────────────────────────────┘
                    ↓
        ┌──────────────────────────────────┐
        │ Continue API request with new jwt│
        └──────────────────────────────────┘

LOGOUT PROCESS:
┌────────────────────────────────────────┐
│  User clicks "Sign Out" in AccountPanel│
└────────────────────────────────────────┘
                    ↓
        ┌──────────────────────────────┐
        │  ClearUserAuth()             │
        │  localStorage.removeItem()   │
        └──────────────────────────────┘
                    ↓
        ┌──────────────────────────────┐
        │  Navigate to /login          │
        └──────────────────────────────┘
```

---

## 2. Game Session Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                   GAME SESSION LIFECYCLE                         │
└──────────────────────────────────────────────────────────────────┘

GAME CREATION:
┌──────────────────────────────────────┐
│  Home Page (/)                       │
│  - Router loader: GET /games         │
│  - Displays list of user's games     │
└──────────────────────────────────────┘
         │
         │ User clicks "Create Game"
         ↓
┌──────────────────────────────────────┐
│  Open Dialog                         │
│  - Input: Character Name             │
└──────────────────────────────────────┘
         │
         │ User submits form
         ↓
┌──────────────────────────────────────┐
│  StartGame() API Call                │
│  POST /games/{randomUUID}            │
│  Body: { playerName: "Hero" }        │
└──────────────────────────────────────┘
         │
         │ Response: { ready: false, error?: null }
         ↓
┌──────────────────────────────────────┐
│  Add new game to list state          │
│  Update UI with new game             │
└──────────────────────────────────────┘

GAME LOAD:
┌──────────────────────────────────────┐
│  Game Page (/game-{$sessionUUID})    │
│  Router loads before rendering       │
│  WorldReady() call                   │
└──────────────────────────────────────┘
         │
         │ GET /worldready/{sessionUUID}
         │ (Polls until ready)
         ↓
┌──────────────────────────────────────┐
│  World generation complete           │
│  Response: { ready: true }           │
└──────────────────────────────────────┘
         │
         ├─→ Check localStorage
         │   gameState-{sessionUUID}
         │   (Get cached state if available)
         │
         └─→ If no cache, fetch fresh
             GET /describe/{sessionUUID}
             Response: {
               game_state: GameState,
               description: string,
               current_room: string,
               rooms: RoomInfo[]
             }
                     ↓
             ┌──────────────────────────────┐
             │ Set gameState in component   │
             │ Set chatHistory from state   │
             │ Cache to localStorage        │
             └──────────────────────────────┘

DURING GAMEPLAY:
┌──────────────────────────────────────┐
│  User types command in Chat input    │
│  Presses Enter or clicks Send        │
└──────────────────────────────────────┘
         │
         ├─→ Add message to chatHistory
         │   (type: "player", content: command)
         │
         └─→ POST /chat/{sessionUUID}
             Body: { chat: "look around" }
                        ↓
             ┌──────────────────────────────┐
             │ Backend processes command    │
             │ Updates game state           │
             │ Returns response             │
             └──────────────────────────────┘
                        ↓
             Response: {
               game_state: GameState (updated)
             }
                        ↓
             ┌──────────────────────────────┐
             │ Update gameState              │
             │ Extract new chat_history     │
             │ Clear input field            │
             │ Cache to localStorage        │
             │ Re-render components         │
             └──────────────────────────────┘

GAME DISPLAY (Parallel Processes):
┌────────────────────────────┬──────────────────────────┐
│   RoomMap Component        │   Chat Component         │
├────────────────────────────┼──────────────────────────┤
│ Receives: gameState        │ Receives: chatHistory    │
│           ↓                │           ↓              │
│ Calculate room positions   │ Render messages          │
│ (force-directed layout)    │ (Markdown rendering)     │
│           ↓                │           ↓              │
│ Draw on Konva canvas       │ Auto-scroll to bottom    │
│ - Circles = rooms          │ - Player messages (blue) │
│ - Player icon (center)     │ - Narrative (gray)       │
│ - Connections (lines)      │                          │
│ - Hover tooltips           │ Input field:             │
│                            │ - Disabled if loading    │
│                            │ - Auto-focus after send  │
└────────────────────────────┴──────────────────────────┘
         ↑                            ↑
         │                            │
         └────────────────────────────┘
             Updates from API
             Every command response

GAME INFO SIDEBAR:
┌──────────────────────────────────────┐
│  GameInfo Component                  │
│  - Current location                  │
│  - Room description                  │
│  - Player inventory                  │
│  - Items in room                     │
│  - NPCs/Occupants                    │
└──────────────────────────────────────┘
         │
         └─→ Updated when gameState changes
             (every chat response)
```

---

## 3. Component Rendering Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                COMPONENT INITIALIZATION & RENDERING              │
└──────────────────────────────────────────────────────────────────┘

APPLICATION START:
main.tsx
    ↓
createRouter() with routeTree
    ↓
<RouterProvider router={router} />
    ↓
┌──────────────────────────────────────────────────┐
│              __root.tsx (Root Layout)            │
│  ┌────────────────────────────────────────────┐  │
│  │ ThemeProvider (MUI)                        │  │
│  │  ↓                                         │  │
│  │ Header Component                          │  │
│  │  - Navigation                             │  │
│  │  - Account Menu                           │  │
│  │                                            │  │
│  │ <Outlet /> → Page-specific component     │  │
│  │                                            │  │
│  │ DevTools (React Query, Router)           │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘

HOME PAGE ROUTING: / (index.tsx)
┌──────────────────────────────────────────┐
│ beforeLoad Hook                          │
│  - isAuthenticated() check               │
│  - Redirect to /login if not auth'd      │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────┐
│ Loader Hook                              │
│  - Async: ListGames() API call           │
│  - GET /games                            │
│  - Returns: ListGamesResponse[]          │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────┐
│ Component Renders                        │
│  - Route.useLoaderData() hooks           │
│  - useState for dialog control           │
│  - MUI components: List, Dialog, etc.    │
│                                          │
│  Events:                                 │
│  - Click game → navigate()               │
│  - Create game → StartGame() + setState()│
└──────────────────────────────────────────┘

GAME PAGE ROUTING: /game-{$sessionUUID}
┌──────────────────────────────────────────┐
│ beforeLoad Hook                          │
│  - isAuthenticated() check               │
│  - Redirect to /login if not auth'd      │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────┐
│ Loader Hook                              │
│  - Async: WorldReady(sessionUUID)        │
│  - Polls GET /worldready/{uuid}          │
│  - Returns: { ready: boolean }           │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────┐
│ Component Renders (PostComponent)        │
│  - Route.useParams() for sessionUUID     │
│  - Route.useLoaderData() for ready flag  │
│                                          │
│  useState Hooks:                         │
│  - gameState (null initially)            │
│  - command (empty string)                │
│  - error (null initially)                │
│  - isLoading (false initially)           │
│  - chatHistory (empty array)             │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────┐
│ useEffect Hook 1: Fetch Game             │
│  - Runs once on mount                    │
│  - pollWorldStatus()                     │
│  - fetchGame():                          │
│    1. DescribeGame() API call            │
│    2. Check localStorage first           │
│    3. Set gameState                      │
│    4. Extract chat_history               │
│    5. Catch & set error                  │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────┐
│ useEffect Hook 2: Save to localStorage   │
│  - Runs when gameState changes           │
│  - localStorage.setItem()                │
│  - Update chatHistory from gameState     │
└──────────────────────────────────────────┘
         ↓
┌──────────────────────────────────────────────────────────────┐
│ Render Game UI Layout (Box containers)                      │
├─────────────────────────────────┬──────────────────────────┤
│      LEFT SIDEBAR               │    MAIN PANEL            │
│  ┌──────────────────────────┐   │ ┌────────────────────┐   │
│  │ RoomMap Component        │   │ │ Chat Component     │   │
│  │ (React Konva)            │   │ │                    │   │
│  │ Props: gameState         │   │ │ Props:             │   │
│  │                          │   │ │ - chatHistory      │   │
│  │ useEffect:               │   │ │ - command          │   │
│  │ - Calculate positions    │   │ │ - setCommand       │   │
│  │ - Memoize calculation    │   │ │ - handleCommand    │   │
│  │                          │   │ │ - isLoading        │   │
│  │ State:                   │   │ │                    │   │
│  │ - hoveredRoom            │   │ │ State:             │   │
│  │ - tooltip                │   │ │ - inputRef         │   │
│  │                          │   │ │                    │   │
│  │ Events:                  │   │ │ Events:            │   │
│  │ - onHover → tooltip      │   │ │ - onKeyPress       │   │
│  │ - onLeave → hide         │   │ │ - onClick (send)   │   │
│  │ - onClick → TBD          │   │ │ - onChange         │   │
│  └──────────────────────────┘   │ └────────────────────┘   │
│                                  │                          │
│  ┌──────────────────────────┐   │ useEffect:                │
│  │ GameInfo Component       │   │ - Auto-scroll to bottom   │
│  │                          │   │ - Focus input on message  │
│  │ Props: gameState         │   │                          │
│  │                          │   │ Render:                  │
│  │ Displays:                │   │ - Message list           │
│  │ - Current room ID        │   │   (ChatMessage sub-comp) │
│  │ - Room description       │   │ - Input field            │
│  │ - Inventory items        │   │ - Send button            │
│  │ - Room items             │   │                          │
│  │ - Occupants              │   │ onClick handlers:        │
│  │                          │   │ - handleCommand()        │
│  │ Events:                  │   │ - handleSubmit()         │
│  │ - Item click → TBD       │   │                          │
│  └──────────────────────────┘   │                          │
│                                  │                          │
└─────────────────────────────────┴──────────────────────────┘

┌──────────────────────────────────┐
│ Error Alert (conditional)        │
│  - Shows if error state set      │
│  - Severity: "error"             │
│  - Dismissible                   │
└──────────────────────────────────┘
```

---

## 4. State Update Lifecycle

```
┌──────────────────────────────────────────────────────────────────┐
│           STATE UPDATES & RE-RENDER CYCLE (Game Page)            │
└──────────────────────────────────────────────────────────────────┘

USER ACTION: Send Command
    ↓
handleCommand() triggered
    ↓
    ├─→ Check if command is empty
    │   return (no update)
    │
    ├─→ setIsLoading(true) ──→ Re-render
    │   (Chat button disabled, spinner shows)
    │
    ├─→ handleSetChatHistory()
    │   setChatHistory([...prev, { type: "player", content }])
    │   ──→ Re-render (Message appears in chat)
    │
    └─→ Chat(sessionUUID, { chat: command })
        POST /chat/{sessionUUID}
        (Network request)
                ↓
        Response: { game_state: GameState }
                ↓
        setGameState(game_state) ──→ Re-render
        (All child components update)
                ↓
        setCommand("") ──→ Re-render
        (Input cleared)
                ↓
        setIsLoading(false) ──→ Re-render
        (Button enabled again)

SIDE EFFECTS OF gameState UPDATE:

useEffect(() => {
  if (!gameState) return;
  
  localStorage.setItem(
    `gameState-${sessionUUID}`,
    JSON.stringify(gameState)
  ); // Persisted to disk
  
  setChatHistory(gameState?.chat_history ?? []);
  // Update chat display
}, [gameState])

CASCADING RE-RENDERS:

setGameState() 
    ↓
useEffect triggers
    ↓
localStorage.setItem() + setChatHistory()
    ↓
RoomMap component re-renders
    ├─→ useMemo(calculateRoomPositions)
    ├─→ Konva canvas updates
    └─→ May trigger room hover state updates
    
GameInfo component re-renders
    ├─→ Displays new room info
    ├─→ Updates inventory
    └─→ Updates occupants list

Chat component re-renders
    ├─→ chatHistory prop updated
    ├─→ New messages appear
    └─→ useEffect: auto-scroll to bottom

PERFORMANCE OPTIMIZATIONS:

1. useMemo in RoomMap
   ├─→ Caches room position calculations
   └─→ Only recalculates if gameState changes

2. useRef for input focus
   ├─→ inputRef.current?.focus()
   └─→ No re-render needed

3. Route loader pattern
   ├─→ Fetch before render
   └─→ No empty state flash

4. localStorage caching
   ├─→ Instant load on revisit
   └─→ Hybrid fresh/cached loading
```

---

## 5. API Request/Response Cycle

```
┌──────────────────────────────────────────────────────────────────┐
│              API REQUEST/RESPONSE FLOW WITH AUTH                 │
└──────────────────────────────────────────────────────────────────┘

COMPONENT CALLS API FUNCTION:
Chat(sessionUUID, { chat: "move north" })
    ↓
api.game.ts: Chat()
    │
    ├─→ POST<ApiChatResponse>(`chat/${sessionUUID}`, reqBody)
    │
    └─→ Return response.data

api.service.ts: POST<T>()
    │
    ├─→ Get auth headers: getAuthHeaders()
    │   │
    │   └─→ auth.service.ts: getAuthHeaders()
    │       ├─→ Get JWT from localStorage
    │       ├─→ Check expiry
    │       │   └─→ If expired: throw redirect to /login
    │       ├─→ Check 30-min buffer
    │       │   └─→ If within buffer: refreshToken()
    │       │       └─→ POST /refresh
    │       │           └─→ Update localStorage with new JWT
    │       └─→ Return AxiosHeaders with Bearer token
    │
    └─→ axios.post(
        ${APP_URI}/${uri},
        body,
        { headers }
      )
      
API_URI = VITE_APP_URI || "http://localhost:8080"

NETWORK REQUEST:
POST http://localhost:8080/chat/{sessionUUID}
Headers: {
  Authorization: "Bearer eyJhbGc..."
}
Body: {
  chat: "move north"
}
        ↓
BACKEND PROCESSING:
    - Parse request
    - Update game state
    - Process narrative
    - Generate AI response
        ↓
API RESPONSE:
200 OK
{
  game_state: {
    current_room: {...},
    player: {...},
    chat_history: [{...}],
    rooms: {...},
    ...
  }
}
        ↓
RESPONSE HANDLING:
axios.post() returns AxiosResponse<ApiChatResponse>
    ↓
response.status (200, 4xx, 5xx)
    ↓
if (response.status > 299):
    console.error("server returned error response")
    ↓
return response
    ↓
Chat API function
    └─→ return response.data
        ↓
Component: const chat = await Chat(...)
    ├─→ chat.game_state available
    ├─→ setGameState(chat.game_state)
    └─→ Triggers re-render cycle

ERROR HANDLING:

try {
  const chat = await Chat(sessionUUID, { chat: command })
  setGameState(chat.game_state)
  setCommand("")
} catch (err: unknown) {
  console.error("Error processing command:", err)
  setError("Failed to process command. Please try again.")
} finally {
  setIsLoading(false)
}

Error types:
- Network error (no server)
- 4xx error (invalid request)
- 5xx error (server error)
- Timeout
- JSON parse error
- Token expired (redirects to login)
```

---

## 6. Authentication Guard Flow

```
┌──────────────────────────────────────────────────────────────────┐
│            ROUTE PROTECTION & AUTHENTICATION GATES               │
└──────────────────────────────────────────────────────────────────┘

USER NAVIGATES TO PROTECTED ROUTE: /game-{$sessionUUID}

beforeLoad Hook Executes:
beforeLoad: async ({ location }) => {
  if (!isAuthenticated()) {
    throw redirect({
      to: "/login",
      search: { redirect: location.href }
    })
  }
}
        ↓
isAuthenticated() Function:
auth.service.ts: isAuthenticated()
    ├─→ tokens = getJWT()
    │   └─→ localStorage.getItem("AAA_JWT")
    │
    ├─→ if (!tokens) return false
    │
    └─→ return tokens.expiresAt > Date.now()
            ↓
        true: Token is valid
        false: Token expired or missing

IF NOT AUTHENTICATED:
    ↓
throw redirect({
  to: "/login",
  search: { redirect: "/game-{uuid}" }
})
    ↓
Router handles redirect
    ↓
User sees login page with redirect param
    ↓
After login success:
navigate({ to: "/" }) or use redirect search param
    ↓
Back to game page, now authenticated

IF AUTHENTICATED:
    ↓
Loader Hook Executes:
loader: async ({ params }) => WorldReady(params.sessionUUID)
    │
    └─→ GET /worldready/{sessionUUID}
        Returns: { ready: boolean }
                ↓
Router waits for loader completion
    ↓
Component renders
    ↓
useEffect hooks fetch additional data

ALL API REQUESTS INCLUDE AUTH HEADER:

getAuthHeaders() called for every API request:
    ├─→ Get JWT from localStorage
    ├─→ Validate expiry
    │   ├─→ Expired? → Redirect to /login
    │   └─→ Within 30-min buffer? → Auto-refresh
    └─→ Return Authorization: "Bearer {jwt}"

TOKEN REFRESH AUTOMATIC FLOW:

Guard in getAuthHeaders():
if (localJWT.expiresAt.valueOf() + 30 * 60 * 1_000 < Date.now().valueOf()) {
    const rHeaders = new AxiosHeaders(`Bearer ${localJWT.rtoken}`)
    refreshToken(rHeaders)
}
        ↓
POST /refresh with refresh_token
        ↓
Response: { token: "new_jwt" }
        ↓
Update localStorage:
localJWT.rtoken = response.data.token
localStorage.setItem("AAA_JWT", JSON.stringify(localJWT))
        ↓
Continue original request with new token

USER LOGS OUT:

handleLogout() in AccountPanel:
    ├─→ ClearUserAuth()
    │   └─→ localStorage.removeItem("AAA_JWT")
    │
    └─→ navigate({ to: "/login" })
            ↓
All stored tokens cleared
            ↓
Next API request:
    ├─→ getJWT() returns undefined
    └─→ Redirect to /login triggered
```

---

## 7. Component Props & Data Flow

```
┌──────────────────────────────────────────────────────────────────┐
│           PARENT → CHILD PROPS DATA FLOW TREE                    │
└──────────────────────────────────────────────────────────────────┘

PostComponent (Game Page State Container)
│
├─ gameState: GameState | null ──────→ RoomMap
│                          │
│                          └──────────→ GameInfo
│
├─ chatHistory: ChatMessageType[] ──→ Chat (parent)
│
├─ command: string ──────────────────→ Chat (parent)
│
├─ setCommand: (cmd: string) => void ──→ Chat (parent)
│   chat input onChange handler
│
├─ isLoading: boolean ───────────────→ Chat (parent)
│   button & input disabled state
│
├─ error: string | null ────────────→ Alert component
│   conditional rendering
│
└─ handleCommand: () => void ────────→ Chat (parent)
   onClick & onKeyPress handlers

RoomMap Component:
├─ Props:
│  └─ gameState: GameState
│     ├─ current_room.id
│     ├─ current_room.connections
│     ├─ rooms (all rooms map)
│     └─ (used to calculate positions)
│
└─ Local State:
   ├─ viewportWidth (window resize)
   ├─ calculatedValue (layout param)
   ├─ hoveredRoom (hover state)
   └─ tooltip (hover tooltip)

GameInfo Component:
├─ Props:
│  ├─ gameState: GameState
│  │  ├─ current_room
│  │  │  ├─ description
│  │  │  ├─ items[]
│  │  │  └─ occupants[]
│  │  └─ player.inventory[]
│  │
│  └─ onItemClick: (item: string) => void
│     (callback for future item interactions)
│
└─ No local state (presentational)

Chat Component:
├─ Props:
│  ├─ chatHistory: ChatMessageType[]
│  │  ├─ Mapped to ChatMessage sub-components
│  │  └─ Markdown rendering
│  │
│  ├─ command: string
│  │  └─ TextField value prop
│  │
│  ├─ setCommand: (cmd: string) => void
│  │  └─ TextField onChange handler
│  │
│  ├─ handleCommand: () => void
│  │  ├─ Button onClick
│  │  └─ TextField onKeyPress ("Enter")
│  │
│  └─ isLoading: boolean
│     ├─ Button disabled state
│     ├─ Input disabled state
│     └─ Shows spinner in button
│
├─ Local State:
│  ├─ chatContainerRef (useRef)
│  │  └─ Scroll position control
│  │
│  └─ inputRef (useRef)
│     └─ Auto-focus after send
│
└─ useEffect Hook:
   └─ Auto-scroll to bottom on chatHistory update

ChatMessage Sub-Component:
└─ Props:
   └─ message: ChatMessageType
      ├─ type: "player" | "narrative"
      └─ content: string
```

---

## 8. State Persistence Strategy

```
┌──────────────────────────────────────────────────────────────────┐
│         DATA PERSISTENCE & CACHING ARCHITECTURE                  │
└──────────────────────────────────────────────────────────────────┘

AUTHENTICATION DATA:
┌──────────────────────────────────────────┐
│ localStorage: AAA_JWT                    │
│ {                                        │
│   jwt: string          (1 hour lifetime) │
│   rtoken: string       (refresh token)   │
│   expiresAt: number    (timestamp)       │
│ }                                        │
│                                          │
│ Lifecycle:                               │
│ - Set on login/signup                    │
│ - Updated on token refresh               │
│ - Cleared on logout                      │
│ - Checked on every API request           │
│ - Expires automatically                  │
└──────────────────────────────────────────┘

GAME STATE DATA:
┌──────────────────────────────────────────┐
│ localStorage: gameState-{sessionUUID}    │
│ {                                        │
│   current_room: Area                     │
│   player: Character                      │
│   visible_items: {...}                   │
│   visible_npcs: {...}                    │
│   connected_rooms: [...]                 │
│   rooms: {...}                           │
│   chat_history: [{...}]                  │
│ }                                        │
│                                          │
│ Lifecycle:                               │
│ - First load: check cache                │
│ - If found: instantly render             │
│ - Then fetch fresh from API              │
│ - Every update: save to localStorage     │
│ - Persists across browser refreshes      │
│ - Manual delete if session deleted       │
└──────────────────────────────────────────┘

LOADING STRATEGY:

Page Load → Game Page
    ↓
1. Router beforeLoad → Check auth token
   ├─→ Valid: continue
   └─→ Expired: redirect to /login
       ↓
2. Router loader → WorldReady() poll
   └─→ Wait for world generation
       ↓
3. Component render
   └─→ Display loading spinner
       ↓
4. useEffect Hook 1:
   ├─→ Check localStorage gameState
   │   ├─→ Found: setState (INSTANT)
   │   │   └─→ UI shows cached version
   │   │
   │   └─→ Not found: null state
   │
   ├─→ API: DescribeGame()
   │   └─→ Fetch fresh from server
   │
   └─→ setState new gameState
       └─→ RE-RENDER with fresh data

BENEFITS:
- Instant display of last known state
- Server fetch happens in parallel
- No loading screen if cache available
- Always synced with server

CHAT HISTORY PERSISTENCE:

Combined approach:
├─→ Stored in gameState.chat_history
│   (saved to localStorage)
│
├─→ Displayed in chatHistory state
│   (component state)
│
└─→ Updated via Chat API response
    (server provides new messages)

Initial load:
1. LocalStorage gameState → chatHistory state
2. Server response → merge with gameState
3. All messages in one history

Across game session:
1. User sends command
2. Response updates gameState
3. useEffect saves to localStorage
4. chatHistory state updated
5. Display updates
6. On page refresh → cache loads

CACHE INVALIDATION:

Game state cache cleared when:
├─→ New command processed
│   └─→ Immediately updated with response
│
├─→ Game deleted
│   └─→ localStorage key removed
│
└─→ Manual page refresh
    └─→ Still loads cache, then fetches fresh

No TTL (time-to-live):
├─→ Cache persists indefinitely
├─→ Server is source of truth
└─→ Stale cache OK (will be updated on API call)
```

---

## Summary

The React client uses a **three-tier data architecture**:

1. **Component State** (React hooks)
   - Real-time UI state
   - Form inputs
   - Loading/error states

2. **localStorage Persistence**
   - Authentication tokens
   - Game state cache
   - Survives page refresh

3. **Backend API**
   - Source of truth
   - Authoritative game state
   - Real-time updates via responses

Data flows from **API → Component → localStorage**,
with **localStorage → Component** on page load.

All API requests are **guarded by authentication checks**,
with **automatic token refresh** before expiration.
