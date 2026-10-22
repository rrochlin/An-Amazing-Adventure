# Phase 6 Architecture Diagram

---

## 1. System Layers

High-level left-to-right view of the four tiers. Edges show the primary flow direction only — detail is in the diagrams below.

```mermaid
graph LR
    subgraph Client["Browser"]
        UI["React App"]
        Admin["Admin Panel"]
    end

    subgraph Edge["AWS Edge"]
        CF["CloudFront"]
        HTTPAPI["HTTP API Gateway"]
        WSAPI["WebSocket API Gateway"]
    end

    subgraph Compute["Lambda · Go 1.24 · arm64"]
        HTTP["HTTP Handlers"]
        WS["WebSocket Handlers"]
        Async["Async Workers"]
    end

    subgraph Data["Data & AI"]
        DDB["DynamoDB"]
        Cog["Cognito"]
        BR["Bedrock"]
        TK["rpg-toolkit"]
    end

    UI --> CF
    UI --> WSAPI
    Admin --> HTTPAPI
    CF --> HTTPAPI
    HTTPAPI --> HTTP
    WSAPI --> WS
    HTTP --> DDB
    HTTP --> Cog
    HTTP --> Async
    WS --> DDB
    WS --> BR
    WS --> TK
    Async --> DDB
    Async --> BR
    Async --> TK
    Cog --> Async
```

---

## 2. HTTP API Routes

```mermaid
graph LR
    Client["Client"]

    subgraph APIGW["API Gateway HTTP · JWT Authorizer"]
        R1["GET /api/games"]
        R2["POST /api/games"]
        R3["DELETE /api/games/{uuid}"]
        R4["PUT /api/users"]
        R5["GET /api/admin/*"]
        R6["PUT /api/admin/users/{id}"]
        R7["POST /api/invites"]
        R8["GET /api/invites/{code}"]
        R9["POST /api/invites/{code}/join"]
    end

    LG["http-games λ"]
    LU["http-users λ"]
    LA["http-admin λ"]
    LI["http-invites λ"]

    Client --> R1 & R2 & R3 --> LG
    Client --> R4 --> LU
    Client --> R5 & R6 --> LA
    Client --> R7 & R8 & R9 --> LI

    LG --> Sessions[("sessions")]
    LG --> Members[("memberships")]
    LG --> Users[("users")]
    LG --> WG["world-gen λ"]
    LA --> Users
    LA --> Cognito[("Cognito")]
    LI --> Invites[("invites")]
    LI --> Sessions
    LI --> Members
```

---

## 3. WebSocket Message Flow

```mermaid
graph LR
    C1["Client 1"]
    C2["Client 2"]

    WSAPI["WebSocket API Gateway"]

    subgraph Handlers["Lambda Handlers"]
        WC["ws-connect"]
        WD["ws-disconnect"]
        WCH["ws-chat"]
        WA["ws-game-action"]
    end

    subgraph Storage["DynamoDB"]
        Conn[("connections")]
        Sess[("sessions")]
        Users[("users")]
        Mut[("mutations")]
    end

    subgraph AI["AI / Rules"]
        Narr["Bedrock Sonnet"]
        Eng["Bedrock Haiku"]
        Toolkit["rpg-toolkit"]
    end

    C1 -->|"$connect"| WSAPI
    C2 -->|"$connect"| WSAPI
    WSAPI --> WC & WD
    WSAPI -->|chat| WCH
    WSAPI -->|game_action| WA

    WC --> Conn & Sess
    WCH --> Sess & Conn & Users & Mut
    WCH --> Narr & Eng & Toolkit
    WA --> Sess & Conn & Toolkit

    WCH -->|narrative + state_delta| C1
    WCH -->|broadcast| C2
    WA -->|state_delta broadcast| C1 & C2
```

---

## 4. Auth & User Lifecycle

```mermaid
sequenceDiagram
    actor User
    participant Client
    participant Cognito
    participant PostConfirm as post-confirm λ
    participant UsersDB as users table
    participant AdminAPI as http-admin λ

    User->>Client: Sign up + confirm email
    Client->>Cognito: signUp() + confirmRegistration()
    Cognito->>PostConfirm: Post-Confirmation trigger
    PostConfirm->>UsersDB: PutUser(role=restricted, ai_enabled=false)

    User->>Client: Log in
    Client->>Cognito: authenticateUser() SRP
    Cognito-->>Client: accessToken + idToken

    Note over Client,UsersDB: User is restricted — AI blocked
    Client->>Client: Decode idToken, role=restricted
    Client->>Client: Show Preview Mode banner

    Note over AdminAPI,UsersDB: Admin approves user
    AdminAPI->>UsersDB: UpdateUser(ai_enabled=true, role=user)
    AdminAPI->>Cognito: AdminAddUserToGroup(user)

    Note over Client,UsersDB: User re-logs in — now approved
    Client->>Cognito: authenticateUser()
    Cognito-->>Client: new token with groups=[user]
```

---

## 5. Party Session Lifecycle

```mermaid
sequenceDiagram
    actor Owner
    actor Player2
    participant API as HTTP API
    participant WS as WebSocket API
    participant DB as DynamoDB

    Owner->>API: POST /api/games
    API->>DB: PutSession + PutMembership(owner)
    API->>API: Invoke world-gen async
    API-->>Owner: session_id

    Owner->>API: POST /api/invites
    API-->>Owner: code "AX7K2M"
    Owner->>Player2: Share /join/AX7K2M

    Player2->>API: GET /api/invites/AX7K2M
    API-->>Player2: game_title, party info
    Player2->>API: POST /api/invites/AX7K2M/join
    API->>DB: PutMembership(player2)
    API-->>Player2: session_id

    Player2->>API: POST /api/games/{id}/join-character
    API->>DB: UpdateSession Players[player2]=char
    API-->>Player2: session_id

    Owner->>WS: $connect
    Player2->>WS: $connect
    Note over WS,DB: Both connections stored with game_id

    Owner->>WS: action=chat
    WS->>DB: GetConnectionsByGameID
    WS-->>Owner: narrative_chunk + state_delta
    WS-->>Player2: narrative_chunk + state_delta
```

---

## 6. Combat Resolution (Phase 3)

```mermaid
sequenceDiagram
    actor Player
    participant WS as WebSocket API
    participant Lambda as ws-game-action λ
    participant Toolkit as rpg-toolkit
    participant DB as DynamoDB
    participant Narr as Bedrock Narrator

    Player->>WS: game_action type=attack
    WS->>Lambda: invoke
    Lambda->>DB: GetSession
    Lambda->>Toolkit: Reconstruct EventBus + rebind conditions
    Lambda->>Toolkit: combat.ResolveAttack(player, skeleton, longsword)
    Note over Toolkit: d20 + STR mod vs AC, 1d8+STR damage
    Toolkit-->>Lambda: AttackResult{hit, damage, targetHP}
    Lambda->>Toolkit: skeleton.TakeTurn()
    Toolkit-->>Lambda: MonsterResult{hit, damage}
    Lambda->>DB: SaveSession with PendingCombatContext
    Lambda-->>Player: state_delta broadcast to all party

    Player->>WS: action=chat
    WS->>Narr: NarrateStream with COMBAT LOG in prompt
    Narr-->>Player: Dramatic narrative of resolved combat
```

---

## 7. World Generation Pipeline (Phase 4)

```mermaid
flowchart TD
    A([POST /api/games]) --> B[Create session\nReady=false]
    B --> C[Invoke world-gen async]
    C --> D([Return session_id\nClient shows terminal])

    subgraph WG["world-gen λ · async · up to 15 min"]
        direction TB
        E["tools/environments - Generate room layout"]
        F["tools/spawn - Place monsters"]
        G["Place key items"]
        H["Bedrock Sonnet - Title · Theme · QuestGoal · Opening"]
        I["BuildDungeonData - SchemaVersion 4"]
        J["DynamoDB\nReady=true"]
        K["Broadcast world_gen_ready"]
        E --> F --> G --> H --> I --> J --> K
    end

    C --> E
    K --> L([Game UI loads])
```

---

## 8. RBAC & Token Quota

```mermaid
flowchart LR
    subgraph Roles
        direction TB
        R["restricted - ai_enabled: false"]
        U["user - ai_enabled: true"]
        A["admin - unlimited"]
    end

    subgraph Capabilities
        direction TB
        C1["Preview game (no AI)"]
        C2["Full AI narration + Party"]
        C3["Admin panel"]
    end

    R --> C1
    U --> C1 & C2
    A --> C1 & C2 & C3

    subgraph QuotaCheck["ws-chat quota check"]
        direction TB
        Q1{"ai_enabled?"}
        Q2{"at token limit?"}
        Q3["Call Bedrock"]
        Q4["Send error frame"]
        Q5["UpdateUserTokens atomic"]
        Q1 -->|no| Q4
        Q1 -->|yes| Q2
        Q2 -->|yes| Q4
        Q2 -->|no| Q3
        Q3 --> Q5
    end

    subgraph Future["billing_mode extension point"]
        direction TB
        F1["admin_granted (now)"]
        F2["own_key (BYOK)"]
        F3["subscription (Stripe)"]
    end
```

---

## 9. Data Model

```mermaid
erDiagram
    USERS {
        string user_id PK
        string role
        int ai_enabled
        int token_limit
        int tokens_used
        int games_limit
        string billing_mode
    }
    SESSIONS {
        string session_id PK
        string owner_id
        int schema_version
        string players_data
        string dungeon_data
        int ready
        string invite_code
    }
    MEMBERSHIPS {
        string user_id PK
        string session_id FK
        string role
    }
    CONNECTIONS {
        string connection_id PK
        string user_id
        string game_id
        int streaming
    }
    INVITES {
        string code PK
        string session_id
        int expires_at
        int uses
    }
    MUTATIONS {
        string session_id PK
        int ts
        string tool
        string input
    }

    USERS ||--o{ MEMBERSHIPS : "member of"
    SESSIONS ||--o{ MEMBERSHIPS : "has members"
    SESSIONS ||--o{ CONNECTIONS : "has connections"
    SESSIONS ||--o{ MUTATIONS : "audit log"
    SESSIONS ||--o| INVITES : "active invite"
    USERS ||--o{ CONNECTIONS : "connected via"
```
