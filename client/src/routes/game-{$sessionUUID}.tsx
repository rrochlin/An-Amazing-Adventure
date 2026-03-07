import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Alert, Box, Button, CircularProgress, Paper, Typography } from "@mui/material";
import { RoomMap } from "../components/RoomMap";
import { GameInfo } from "../components/GameInfo";
import { Chat } from "../components/Chat";
import { isAuthenticated } from "../services/auth.service";
import { LoadGame, WorldReady } from "../services/api.game";
import { pollWorldStatus } from "@/components/WaitForWorld";
import { useGameStore } from "../store/gameStore";
import { useGameSocket } from "../hooks/useGameSocket";
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
  loader: async ({ params }) => WorldReady(params.sessionUUID),
});

function GamePage() {
  const { sessionUUID } = Route.useParams();
  const [command, setCommand] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loadingGame, setLoadingGame] = useState(true);

  const { gameState, chatMessages, streamingMessage, isStreaming, wsError, wsStatus, addChatMessage, setGameState, reset } =
    useGameStore();

  const { sendChat, sendAction } = useGameSocket({ sessionId: sessionUUID, enabled: !!gameState });

  // Load game state on mount (poll world-ready then fetch full state)
  useEffect(() => {
    reset();
    let cancelled = false;
    const init = async () => {
      const ready = await pollWorldStatus(sessionUUID);
      if (cancelled) return;
      if (!ready) {
        setLoadError("World generation timed out — please refresh or create a new game.");
        setLoadingGame(false);
        return;
      }
      try {
        const data = await LoadGame(sessionUUID);
        if (!cancelled) {
          setGameState(data.state);
        }
      } catch {
        if (!cancelled) setLoadError("Failed to load game — please try again.");
      } finally {
        if (!cancelled) setLoadingGame(false);
      }
    };
    init();
    return () => { cancelled = true; };
  }, [sessionUUID]);

  const handleCommand = () => {
    if (!command.trim() || isStreaming) return;
    addChatMessage({ type: "player", content: command });
    sendChat(command);
    setCommand("");
  };

  if (loadingGame) {
    return (
      <Box sx={{ p: 4, textAlign: "center" }}>
        <CircularProgress />
        <Typography sx={{ mt: 2 }}>Loading adventure...</Typography>
      </Box>
    );
  }

  if (loadError) {
    return (
      <Box sx={{ p: 4 }}>
        <Alert severity="error">{loadError}</Alert>
      </Box>
    );
  }

  // Combine committed messages with any in-flight streaming message
  const displayMessages = streamingMessage
    ? [...chatMessages, { type: "narrative" as const, content: streamingMessage }]
    : chatMessages;

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
          <Typography variant="h6" sx={{
            mb: 2, textAlign: "center", textTransform: "uppercase",
            letterSpacing: "0.1em", borderBottom: `2px solid ${AppTheme.palette.primary.main}`, pb: 1,
          }}>
            World Map
          </Typography>
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
