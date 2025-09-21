import { Button, Typography, Box } from "@mui/material";
import { type GameState } from "../types/types";

export const GameInfo = ({
  gameState,
  onItemClick,
}: {
  gameState: GameState;
  onItemClick: (item: string) => void;
}) => {
  const currentRoom = gameState.current_room;

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="h6">
        Current Location: {gameState.current_room.id}
      </Typography>
      <Typography variant="body1">
        {currentRoom?.description || "No description available"}
      </Typography>

      <Typography variant="h6" sx={{ mt: 2 }}>
        Your Inventory:
      </Typography>
      {gameState.player.inventory && gameState.player.inventory.length > 0 ? (
        <ul>
          {gameState.player.inventory.map((item, idx) => (
            <li key={idx}>
              <Button
                onClick={() => onItemClick(item.name)}
                sx={{ textTransform: "none", color: "#2196F3" }}
              >
                {item.name}
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <Typography>Your inventory is empty</Typography>
      )}

      <Typography variant="h6" sx={{ mt: 2 }}>
        Items in Room:
      </Typography>
      {currentRoom?.items && currentRoom.items.length > 0 ? (
        <ul>
          {currentRoom.items.map((item, idx) => (
            <li key={10 * idx}>
              <Button
                onClick={() => onItemClick(item.name)}
                sx={{ textTransform: "none", color: "#2196F3" }}
              >
                {item.name}
              </Button>
            </li>
          ))}
        </ul>
      ) : (
        <Typography>No items in this room</Typography>
      )}

      <Typography variant="h6" sx={{ mt: 2 }}>
        Occupants:
      </Typography>
      {currentRoom?.occupants && currentRoom.occupants.length > 0 ? (
        <ul>
          {currentRoom.occupants.map((occupant) => (
            <li key={occupant}>{occupant}</li>
          ))}
        </ul>
      ) : (
        <Typography>No occupants in this room</Typography>
      )}
    </Box>
  );
};
