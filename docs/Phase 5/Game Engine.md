## Entities

```go
type Room struct {
    ID          uuid.UUID
    Name        string
    Description string
    Connections map[string]uuid.UUID  // direction → room ID
    Coordinates Coordinates
    Items       []Item
    Occupants   []uuid.UUID           // character IDs
}

type Character struct {
    ID          uuid.UUID
    Name        string
    Backstory   string
    Description string
    Location    uuid.UUID             // room ID
    Backpack     Inventory
    Equipment   Equipment
    Alive       bool
    Health      int                  // 0–100
    Friendly    bool
}

type Item struct {
    ID          uuid.UUID
    Name        string
    Description string
    Weight      float64
    Equippable  bool
    Slot        EquipmentSlot        // empty string if not equippable
}

type Container struct {
    ID          uuid.UUID
    Name        string
    Description string
    Location    uuid.UUID            // room ID or character ID (if in inventory)
    Contents    Inventory
    Locked      bool
    KeyItemID   *uuid.UUID           // nil if not locked
}
```

## Abstracts

```go
type Inventory struct {
    Items      []Item
    Containers []Container
}

type Equipment struct {
    Head   *Item
    Chest  *Item
    Legs   *Item
    Hands  *Item
    Feet   *Item
    Back   *Item
}

type EquipmentSlot string
const (
    SlotHead  EquipmentSlot = "head"
    SlotChest EquipmentSlot = "chest"
    SlotLegs  EquipmentSlot = "legs"
    SlotHands EquipmentSlot = "hands"
    SlotFeet  EquipmentSlot = "feet"
    SlotBack  EquipmentSlot = "back"
)

type Coordinates struct {
    X, Y, Z float64
}
```

## Game State

```go
type Game struct {
    GameID        uuid.UUID
    UserID        uuid.UUID
    SchemaVersion int                       // increment when struct changes
    Player        Character
    Map           map[uuid.UUID]Room
    Items         map[uuid.UUID]Item
    NPCs          map[uuid.UUID]Character
    Containers    map[uuid.UUID]Container
    Narrative     []types.Message           // Bedrock conversation history
    Ready         bool
    CreatedAt     time.Time
}
```

### Schema Versioning
- `SchemaVersion = 1`: initial Bedrock/AWS version.
- On `LoadGame`, check `SchemaVersion`. If lower than current, run a migration function before returning the game.
- Existing DynamoDB records from GCP/Gemini era have no `SchemaVersion` — treat as version 0 (incompatible, return not found).

## Game Engine Actions (WebSocket game_action route)

Non-AI game state mutations that the player can trigger directly from the UI:

- `equip_item(itemID)` — moves item from backpack to the appropriate `Equipment` slot.
- `unequip_item(slot)` — moves item from equipment slot back to backpack.
- `drop_item(itemID)` — moves item from backpack/equipment to current room's items list.
- `pick_up_item(itemID)` — moves item from current room to player's backpack.
- `open_container(containerID)` — returns container contents (no state change unless locked).
- `move(direction)` — moves the player to the connected room in the given direction. Updates `Player.Location`. May trigger a narrative response (server decides).

These are handled by the `ws-game-action` Lambda, not the AI. They return a `game_state_update` WebSocket frame with the new state. The AI is only involved in `chat` route messages.

## AI Tool Set (for Narrator during chat)

The narrator has access to the same world mutation tools during gameplay as the world builder had during generation:

- `create_room(name, description, connectTo, direction)`
- `update_room(id, description)`
- `create_item(name, description, weight, equippable, slot)`
- `create_character(name, description, backstory, friendly, health)`
- `create_container(name, description, location)`
- `set_item_location(itemID, locationID)`
- `set_character_location(characterID, roomID)`
- `connect_rooms(fromID, toID, direction)`
- `get_room_info(roomID)`
- `list_items_in_room(roomID)`
- `list_characters_in_room(roomID)`
- `give_item_to_player(itemID)` — places item in player's backpack
- `remove_item_from_player(itemID)` — removes item from player's backpack (taken, lost, etc.)
- `damage_character(characterID, amount)`
- `heal_character(characterID, amount)`
- `set_character_alive(characterID, alive)`
- `stop_generation` — signals end of response (no more tool calls or narrative expected)

## Concurrency / Locking

With Lambda concurrency, two simultaneous requests to the same game session can cause race conditions on the DynamoDB record.

Strategy: optimistic locking with a `Version` counter on the game state record.
- Add `Version int` to `Game` / `SaveState`.
- `LoadGame` reads the item and notes the current version.
- `SaveGame` uses a DynamoDB condition expression: `attribute_exists(version) AND version = :expectedVersion`.
- On `ConditionalCheckFailedException`, retry the load → mutate → save cycle (max 3 retries).
- The WebSocket `streaming_in_progress` flag in the connections table provides the first line of defense (blocks concurrent chat messages at the application layer before they hit DynamoDB).
