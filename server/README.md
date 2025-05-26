# Text Adventure Game Server

A Go-based server for a text adventure game that uses AI to generate and manage an interactive world. The server provides a REST API for game interactions and uses Google's Gemini AI for dynamic world generation and narrative responses.

## Features

- **AI-Powered World Generation**: Dynamic creation of rooms, items, and NPCs
- **Interactive Narrative**: AI acts as a Dungeon Master, providing contextual responses
- **Room Management**: Create, connect, and navigate between rooms
- **Item System**: Create and manage items with properties
- **NPC System**: Create and manage non-player characters with friendly/hostile states
- **Real-time Game State**: Track player location, inventory, and world state

## Architecture

### Core Components

- **Game Engine** (`game.go`): Manages the core game state and logic
- **World Generator** (`world_gen.go`): Handles AI-driven world generation
- **Area System** (`area.go`): Manages rooms and their connections
- **Character System** (`character.go`): Handles player and NPC management
- **Item System** (`item.go`): Manages game items and their properties
- **MCP System** (`mcp.go`): Manages AI communication and tool execution

### API Endpoints

- `POST /api/startgame`: Initialize a new game session
- `POST /api/move`: Move player between rooms
- `GET /api/describe`: Get current room description
- `POST /api/chat`: Interact with the AI Dungeon Master

## Setup

### Prerequisites

- Go 1.21 or higher
- Google Cloud API key for Gemini AI
- Environment variables:
  - `GCP_KEY`: Your Google Cloud API key
  - `HOST_URL`: Server host (default: localhost)
  - `PORT`: Server port (default: 8080)

### Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Create a `.env` file with your configuration:
   ```
   GCP_KEY=your_api_key_here
   HOST_URL=localhost
   PORT=8080
   ```
4. Run the server:
   ```bash
   go run .
   ```

## Game Flow

1. **Game Start**
   - Client calls `/api/startgame`
   - Server initializes game state
   - AI generates initial world

2. **World Generation**
   - AI creates rooms, items, and NPCs
   - Rooms are connected logically
   - Items and NPCs are placed in appropriate rooms

3. **Player Interaction**
   - Player can move between rooms
   - Player can interact with items and NPCs
   - AI provides narrative responses

4. **State Management**
   - Game state is maintained server-side
   - Player location and inventory are tracked
   - Room contents and connections are managed

## API Usage

### Start Game
```http
POST /api/startgame
```
Response:
```json
{
    "status": "Game started"
}
```

### Move Player
```http
POST /api/move
{
    "room_id": "tavern"
}
```
Response:
```json
{
    "status": "Moved to room tavern"
}
```

### Get Room Description
```http
GET /api/describe
```
Response:
```json
{
    "description": "Room tavern:\n\nItems:\n- Torch\n\nOccupants:\n- Guard\n\nConnections:\n- Room kitchen"
}
```

### Chat with AI
```http
POST /api/chat
{
    "chat": "What do I see in this room?"
}
```
Response:
```json
{
    "response": "You find yourself in a cozy tavern. A warm fire crackles in the hearth, and a friendly guard stands by the door. A torch hangs on the wall, providing additional light."
}
```

## Development

### Project Structure
```
server/
├── main.go          # Server entry point
├── game.go          # Core game logic
├── world_gen.go     # World generation
├── area.go          # Room management
├── character.go     # Character system
├── item.go          # Item system
├── mcp.go           # AI communication
├── handlers.go      # API handlers
├── handler_chat.go  # Chat handler
├── errors.go        # Error handling
└── logger.go        # Logging utilities
```

### Adding New Features

1. **New Game Mechanics**
   - Add methods to `game.go`
   - Update `ExecuteTool` in `mcp.go`
   - Add corresponding API endpoints

2. **New AI Capabilities**
   - Update system instructions in `mcp.go`
   - Add new tools to `ExecuteTool`
   - Update chat handler for new interactions

3. **New API Endpoints**
   - Add handler in `handlers.go`
   - Update main router in `main.go`
   - Add corresponding game logic

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
