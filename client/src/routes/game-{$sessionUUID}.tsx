import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, Box, Button, IconButton, Paper, Tooltip, Typography } from "@mui/material";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
import { RoomMap } from "../components/RoomMap";
import { GameInfo } from "../components/GameInfo";
import { Chat } from "../components/Chat";
import { isAuthenticated } from "../services/auth.service";
import { LoadGame } from "../services/api.game";
import { useGameStore } from "../store/gameStore";
import { useGameSocket } from "../hooks/useGameSocket";
import { WorldGenTerminal } from "../components/WorldGenTerminal";
import { AppTheme } from "@/theme/theme";

function GameErrorFallback() {
  const navigate = useNavigate();
  return (
    <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "calc(100vh - 64px)", p: 4 }}>
      <Paper sx={{ maxWidth: 480, width: "100%", p: 4, textAlign: "center" }}>
        <Typography variant="h5" sx={{ mb: 2, fontFamily: '"Cinzel", serif' }}>
          Adventure Unavailable
        </Typography>
        <Alert severity="error" sx={{ mb: 3, textAlign: "left" }}>
          This adventure could not be loaded. It may have been deleted or the server encountered an error.
        </Alert>
        <Button variant="contained" onClick={() => navigate({ to: "/" })}>
          Return to Adventures
        </Button>
      </Paper>
    </Box>
  );
}

export const Route = createFileRoute("/game-{$sessionUUID}")({
  component: GamePage,
  errorComponent: GameErrorFallback,
  beforeLoad: async ({ location }) => {
    if (!isAuthenticated()) {
      throw redirect({ to: "/login", search: { redirect: location.href } });
    }
  },
});

function GamePage() {
  const { sessionUUID } = Route.useParams();
  const navigate = useNavigate();
  const [command, setCommand] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loadingGame, setLoadingGame] = useState(true);

  const { gameState, chatMessages, streamingMessage, isStreaming, wsError, wsStatus, worldGenLog, worldGenReady, addChatMessage, setGameState, reset } =
    useGameStore();

  // Load game state from the server. Called once on mount, and again when world_gen_ready fires.
  const loadGameRef = useRef(false);
  const loadGame = useCallback(async () => {
    if (loadGameRef.current) return;
    loadGameRef.current = true;
    try {
      const data = await LoadGame(sessionUUID);
      if (data.ready && data.state) {
        setGameState(data.state);
      }
      // If not ready, the WebSocket will deliver world_gen_ready when done
    } catch {
      setLoadError("Failed to load game — please try again.");
    } finally {
      setLoadingGame(false);
    }
  }, [sessionUUID, setGameState]);

  // Called by useGameSocket when world_gen_ready arrives
  const handleWorldReady = useCallback(() => {
    loadGameRef.current = false; // allow reload
    setLoadingGame(true);
    loadGame();
  }, [loadGame]);

  const { sendChat, sendAction } = useGameSocket({
    sessionId: sessionUUID,
    onWorldReady: handleWorldReady,
  });

  useEffect(() => {
    reset();
    loadGameRef.current = false;
    loadGame();
    return () => { /* cleanup handled in useGameSocket */ };
  }, [sessionUUID]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleCommand = () => {
    if (!command.trim() || isStreaming) return;
    addChatMessage({ type: "player", content: command });
    sendChat(command);
    setCommand("");
  };

  if (loadError) {
    return (
      <Box sx={{ p: 4 }}>
        <Alert severity="error">{loadError}</Alert>
      </Box>
    );
  }

  // Show world-gen terminal while world is being built (not yet ready)
  if (loadingGame || (!gameState && !loadError)) {
    return (
      <Box
        sx={{
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          alignItems: "center",
          minHeight: "calc(100vh - 64px)",
          p: 4,
          gap: 3,
        }}
      >
        <Paper
          sx={{
            maxWidth: 600,
            width: "100%",
            background: "rgba(0, 8, 0, 0.96)",
            border: "1px solid rgba(0, 255, 70, 0.25)",
            boxShadow: "0 0 40px rgba(0, 255, 70, 0.15)",
            overflow: "hidden",
          }}
        >
          <Box
            sx={{
              px: 2,
              py: 1.5,
              borderBottom: "1px solid rgba(0, 255, 70, 0.2)",
            }}
          >
            <Typography
              sx={{
                fontFamily: '"Cinzel", "Georgia", serif',
                color: "rgba(0, 255, 70, 0.9)",
                fontSize: "1rem",
              }}
            >
              Forging Your World
            </Typography>
          </Box>
          <Box sx={{ p: 2 }}>
            <WorldGenTerminal lines={worldGenLog} ready={worldGenReady} />
          </Box>
          {worldGenLog.length === 0 && !worldGenReady && (
            <Box sx={{ px: 2, pb: 1.5 }}>
              <Typography
                variant="caption"
                sx={{ color: "rgba(0,255,70,0.4)", fontFamily: "monospace" }}
              >
                Connecting to world generator...
              </Typography>
            </Box>
          )}
        </Paper>
        {worldGenReady && (
          <Alert
            severity="success"
            sx={{
              maxWidth: 600,
              width: "100%",
              background: "rgba(0, 255, 70, 0.08)",
              color: "rgba(0,255,70,0.9)",
              border: "1px solid rgba(0,255,70,0.3)",
            }}
          >
            World ready — loading your adventure...
          </Alert>
        )}
      </Box>
    );
  }

  // displayMessages contains only committed (finalized) messages.
  // The in-flight streamingMessage is passed separately to Chat so it can
  // render a dedicated StreamingChatMessage bubble rather than a plain entry.
  const displayMessages = chatMessages;

  return (
    <Box
      sx={{
        height: `calc(100vh - ${AppTheme.mixins.toolbar.minHeight}px)`,
        display: "flex",
        flexDirection: "row",
        overflow: "hidden",
        backgroundColor: "background.default",
        gap: 2,
        p: 2,
        pr: 3,
        width: "100%",
        maxWidth: "100vw",
        boxSizing: "border-box",
      }}
    >
      {/* Left — Map (25%) */}
      <Box sx={{ flex: "0 0 25%", minWidth: 0, display: "flex", flexDirection: "column", gap: 2 }}>
        <Paper sx={{
          flex: 1, p: 2, display: "flex", flexDirection: "column", overflow: "hidden",
          transition: "all 0.3s ease-in-out",
          "&:hover": { boxShadow: "0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)" },
        }}>
          <Box sx={{ display: "flex", alignItems: "center", mb: 2, borderBottom: `2px solid ${AppTheme.palette.primary.main}`, pb: 1 }}>
            <Typography variant="h6" sx={{
              flex: 1, textAlign: "center", textTransform: "uppercase", letterSpacing: "0.1em",
            }}>
              World Map
            </Typography>
            <Tooltip title="Adventure details">
              <IconButton
                size="small"
                onClick={() => navigate({ to: "/game-{$sessionUUID}/details", params: { sessionUUID } })}
                sx={{ color: "primary.main", opacity: 0.7, "&:hover": { opacity: 1 } }}
              >
                <InfoOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
          <Box sx={{ flex: 1, display: "flex", justifyContent: "center", alignItems: "center" }}>
            <RoomMap gameState={gameState} />
          </Box>
        </Paper>
      </Box>

      {/* Center — Chat (50%) */}
      <Box sx={{ flex: "0 0 50%", minWidth: 0, display: "flex", flexDirection: "column" }}>
        <Paper sx={{
          flex: 1, overflow: "hidden",
          transition: "all 0.3s ease-in-out",
          "&:hover": { boxShadow: "0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)" },
        }}>
          <Chat
            chatHistory={displayMessages}
            streamingMessage={streamingMessage || undefined}
            command={command}
            setCommand={setCommand}
            handleCommand={handleCommand}
            isLoading={isStreaming}
          />
        </Paper>
        {(wsError || wsStatus === "error") && (
          <Alert severity="warning" sx={{ mt: 1 }}>
            {wsError ?? "Connection lost — retrying..."}
          </Alert>
        )}
      </Box>

      {/* Right — Game Info (25%) */}
      <Box sx={{ flex: "0 0 25%", minWidth: 0, display: "flex", gap: 2 }}>
        <Paper sx={{
          flex: 1, overflow: "hidden", display: "flex", flexDirection: "column",
          transition: "all 0.3s ease-in-out",
          "&:hover": { boxShadow: "0 6px 24px rgba(0,0,0,0.6), inset 0 1px 0 rgba(201,169,98,0.2)" },
        }}>
          <GameInfo gameState={gameState} sendAction={sendAction} />
        </Paper>
        <Box sx={{ width: "4px", backgroundColor: "#000", opacity: 0.5, borderRadius: "2px" }} />
      </Box>
    </Box>
  );
}
