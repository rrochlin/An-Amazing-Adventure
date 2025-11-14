import { Button, Typography, Box, Divider, Paper, Tabs, Tab } from "@mui/material";
import { type GameState } from "../types/types";
import { useState } from "react";

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`game-tabpanel-${index}`}
      aria-labelledby={`game-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ py: 2 }}>{children}</Box>}
    </div>
  );
}

export const GameInfo = ({
  gameState,
  onItemClick,
}: {
  gameState: GameState;
  onItemClick: (item: string) => void;
}) => {
  const currentRoom = gameState.current_room;
  const [tabValue, setTabValue] = useState(0);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const SectionHeader = ({ children }: { children: React.ReactNode }) => (
    <Typography
      variant="h6"
      sx={{
        mb: 1,
        textTransform: "uppercase",
        letterSpacing: "0.1em",
        fontSize: "0.9rem",
        borderLeft: "4px solid",
        borderColor: "primary.main",
        pl: 1.5,
      }}
    >
      {children}
    </Typography>
  );

  return (
    <Box sx={{ height: "100%", display: "flex", flexDirection: "column", minWidth: 0 }}>
      <Tabs
        value={tabValue}
        onChange={handleTabChange}
        sx={{
          borderBottom: 1,
          borderColor: "divider",
          minHeight: "40px",
          minWidth: 0,
          "& .MuiTab-root": {
            minHeight: "40px",
            textTransform: "uppercase",
            fontSize: "0.75rem",
            letterSpacing: "0.05em",
            minWidth: 0,
          },
        }}
      >
        <Tab label="Location" />
        <Tab label="Inventory" />
        <Tab label="Room" />
      </Tabs>

      <Box
        sx={{
          flex: 1,
          overflow: "auto",
          px: 2,
          minWidth: 0,
          "&::-webkit-scrollbar": {
            width: "8px",
          },
          "&::-webkit-scrollbar-track": {
            background: "background.default",
            borderRadius: "4px",
          },
          "&::-webkit-scrollbar-thumb": {
            background: "primary.dark",
            borderRadius: "4px",
            transition: "background 0.2s ease-in-out",
            "&:hover": {
              background: "primary.main",
            }
          },
        }}
      >
        {/* Location Tab */}
        <TabPanel value={tabValue} index={0}>
          <Typography
            variant="h6"
            sx={{
              mb: 1,
              textAlign: "center",
              textTransform: "uppercase",
              letterSpacing: "0.1em",
              borderBottom: "2px solid",
              borderColor: "primary.main",
              pb: 1,
              wordWrap: "break-word",
              overflowWrap: "break-word",
            }}
          >
            {gameState.current_room.id.split("_").map(
              (word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()
            ).join(" ")}
          </Typography>

          <Paper
            sx={(theme) => ({
              p: 1.5,
              mt: 2,
              backgroundColor:
                theme.palette.mode === "dark"
                  ? "rgba(201, 169, 98, 0.05)"
                  : "rgba(160, 130, 109, 0.15)",
              border:
                theme.palette.mode === "dark"
                  ? "1px solid rgba(201, 169, 98, 0.2)"
                  : "2px solid rgba(139, 111, 71, 0.4)",
              boxShadow:
                theme.palette.mode === "dark"
                  ? "none"
                  : "0 1px 3px rgba(107, 86, 56, 0.2)",
            })}
          >
            <Typography variant="body2" sx={{ fontStyle: "italic", lineHeight: 1.6, wordWrap: "break-word", overflowWrap: "break-word" }}>
              {currentRoom?.description || "No description available"}
            </Typography>
          </Paper>
        </TabPanel>

        {/* Inventory Tab */}
        <TabPanel value={tabValue} index={1}>
          <SectionHeader>Your Inventory</SectionHeader>
          <Divider
            sx={(theme) => ({
              mb: 1,
              borderColor:
                theme.palette.mode === "dark"
                  ? "rgba(201, 169, 98, 0.3)"
                  : "rgba(139, 111, 71, 0.4)",
            })}
          />
          {gameState.player.inventory && gameState.player.inventory.length > 0 ? (
            <Box component="ul" sx={{ pl: 2, m: 0 }}>
              {gameState.player.inventory.map((item, idx) => (
                <Box component="li" key={idx} sx={{ mb: 0.5 }}>
                  <Button
                    onClick={() => onItemClick(item.name)}
                    sx={{
                      textTransform: "none",
                      color: "primary.main",
                      p: 0,
                      minWidth: 0,
                      fontFamily: "Crimson Text, Georgia, serif",
                      fontSize: "1rem",
                      wordWrap: "break-word",
                      overflowWrap: "break-word",
                      whiteSpace: "normal",
                      textAlign: "left",
                      "&:hover": {
                        color: "primary.light",
                        backgroundColor: "transparent",
                      }
                    }}
                  >
                    {item.name}
                  </Button>
                </Box>
              ))}
            </Box>
          ) : (
            <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic" }}>
              Your inventory is empty
            </Typography>
          )}
        </TabPanel>

        {/* Room Tab */}
        <TabPanel value={tabValue} index={2}>
          <SectionHeader>Items in Room</SectionHeader>
          <Divider
            sx={(theme) => ({
              mb: 1,
              borderColor:
                theme.palette.mode === "dark"
                  ? "rgba(201, 169, 98, 0.3)"
                  : "rgba(139, 111, 71, 0.4)",
            })}
          />
          {currentRoom?.items && currentRoom.items.length > 0 ? (
            <Box component="ul" sx={{ pl: 2, m: 0 }}>
              {currentRoom.items.map((item, idx) => (
                <Box component="li" key={10 * idx} sx={{ mb: 0.5 }}>
                  <Button
                    onClick={() => onItemClick(item.name)}
                    sx={{
                      textTransform: "none",
                      color: "primary.main",
                      p: 0,
                      minWidth: 0,
                      fontFamily: "Crimson Text, Georgia, serif",
                      fontSize: "1rem",
                      wordWrap: "break-word",
                      overflowWrap: "break-word",
                      whiteSpace: "normal",
                      textAlign: "left",
                      "&:hover": {
                        color: "primary.light",
                        backgroundColor: "transparent",
                      }
                    }}
                  >
                    {item.name}
                  </Button>
                </Box>
              ))}
            </Box>
          ) : (
            <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic" }}>
              No items in this room
            </Typography>
          )}

          <SectionHeader>Occupants</SectionHeader>
          <Divider
            sx={(theme) => ({
              mb: 1,
              borderColor:
                theme.palette.mode === "dark"
                  ? "rgba(201, 169, 98, 0.3)"
                  : "rgba(139, 111, 71, 0.4)",
            })}
          />
          {currentRoom?.occupants && currentRoom.occupants.length > 0 ? (
            <Box component="ul" sx={{ pl: 2, m: 0 }}>
              {currentRoom.occupants.map((occupant) => (
                <Box component="li" key={occupant} sx={{ mb: 0.5 }}>
                  <Typography variant="body2" sx={{ fontFamily: "Crimson Text, Georgia, serif", wordWrap: "break-word", overflowWrap: "break-word" }}>
                    {occupant}
                  </Typography>
                </Box>
              ))}
            </Box>
          ) : (
            <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic" }}>
              No occupants in this room
            </Typography>
          )}
        </TabPanel>
      </Box>
    </Box>
  );
};
