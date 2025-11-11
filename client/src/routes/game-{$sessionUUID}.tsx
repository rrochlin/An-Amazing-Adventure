import { createFileRoute, redirect } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import type { ChatMessageType, GameState } from "../types/types";
import {
  Chat as SendChat,
  DescribeGame,
  WorldReady,
} from "../services/api.game";
import { Alert, Box, CircularProgress, Paper, Typography } from "@mui/material";
import { RoomMap } from "../components/RoomMap";
import { GameInfo } from "../components/GameInfo";
import { Chat } from "../components/Chat";
import { isAuthenticated } from "../services/auth.service";
import { AppTheme } from "@/theme/theme";
import { pollWorldStatus } from "@/components/WaitForWorld";

export const Route = createFileRoute("/game-{$sessionUUID}")({
  component: PostComponent,
  beforeLoad: async ({ location }) => {
    if (!isAuthenticated()) {
      throw redirect({
        to: "/login",
        search: { redirect: location.href },
      });
    }
  },
  loader: async ({ params }) => WorldReady(params.sessionUUID),
});

function PostComponent() {
  const { sessionUUID } = Route.useParams();
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [command, setCommand] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [chatHistory, setChatHistory] = useState<ChatMessageType[]>([]);
  const { ready: initialReady } = Route.useLoaderData();

  // loads chat from localStorage. Should default to pull from API if not available
  const handleSetChatHistory = (message: ChatMessageType) => {
    setChatHistory((prevChatHistory) => {
      const newChat = [...prevChatHistory, message];
      return newChat;
    });
  };

  useEffect(() => {
    if (!gameState) return;
    localStorage.setItem(`gameState-${sessionUUID}`, JSON.stringify(gameState));
    setChatHistory(gameState?.chat_history ?? []);
  }, [gameState]);

  useEffect(() => {
    const getGame = async () => {
      const result = await pollWorldStatus(sessionUUID);
      if (result) {
        fetchGame();
      } else {
        setError(
          "World generation is taking a long time please refresh\nif the issue persists create a new game",
        );
        return;
      }
    };
    getGame();
  }, []);

  // This will load gameState into the client. We can write this to localStorage
  // and also verify that it's in sync with the backend
  const fetchGame = async () => {
    setIsLoading(true);
    setError(null);

    try {
      if (!initialReady) {
        // will be true initially if onload worldready call succeeded
        const worldGenerated = await WorldReady(sessionUUID);
        if (!worldGenerated.ready) {
          setError(
            "World generation is taking longer than expected. You can still try to play, but some features might not be available yet.",
          );
          console.log("returned early");
          return;
        }
      }

      // Get initial game state for quick load
      let state: GameState;
      const localState = localStorage.getItem(`gameState-${sessionUUID}`);
      if (localState != null && localState != "null") {
        state = JSON.parse(localState);
        setGameState(state);
      }

      const gameResponse = await DescribeGame(sessionUUID);
      state = gameResponse.game_state;
      setGameState(state);
    } catch (err) {
      setError(
        "Failed to start game. Please check if the server is running and try again.",
      );
      console.error("Error starting game:", err);
    }
    setIsLoading(false);
  };

  // Sends a chat and updates the game
  const handleCommand = async () => {
    if (!command.trim()) return;

    setIsLoading(true);
    setError(null);

    try {
      handleSetChatHistory({ type: "player", content: command });

      const chat = await SendChat(sessionUUID, { chat: command });
      setGameState(chat.game_state);
      setCommand("");
    } catch (err) {
      setError("Failed to process command. Please try again.");
      console.error("Error processing command:", err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleItemClick = async (item: string) => {
    console.log(item);
    return;
  };

  if (!gameState && gameState != "null") {
    return (
      <Box sx={{ p: 4, textAlign: "center" }}>
        <CircularProgress />
        <Typography sx={{ mt: 2 }}>Loading game state...</Typography>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        height: `calc(100vh - ${AppTheme.mixins.toolbar.minHeight}px - 8px)`,
        display: "flex",
        flexDirection: "row",
        overflow: "hidden",
        backgroundColor: "#1E1E1E",
        gap: 2,
        p: 2,
      }}
    >
      {/* Left Sidebar - Map (25%) */}
      <Box sx={{ flex: "0 0 25%", display: "flex", flexDirection: "column", gap: 2 }}>
        <Paper
          sx={{
            flex: 1,
            backgroundColor: "#2D2D2D",
            p: 2,
            display: "flex",
            flexDirection: "column",
            overflow: "hidden"
          }}
        >
          <Typography variant="h6" sx={{ color: "#E0E0E0", mb: 2 }}>
            World Map
          </Typography>
          <Box sx={{ flex: 1, display: "flex", justifyContent: "center", alignItems: "center" }}>
            <RoomMap gameState={gameState} />
          </Box>
        </Paper>
      </Box>

      {/* Center - Chat Area (50%) */}
      <Box sx={{ flex: "0 0 50%", display: "flex", flexDirection: "column" }}>
        <Paper sx={{ flex: 1, backgroundColor: "#2D2D2D", overflow: "hidden" }}>
          <Chat
            chatHistory={chatHistory}
            command={command}
            setCommand={setCommand}
            handleCommand={handleCommand}
            isLoading={isLoading}
          />
        </Paper>
        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}
      </Box>

      {/* Right Sidebar - Game Info (25%) */}
      <Box sx={{ flex: "0 0 25%", display: "flex" }}>
        <Paper
          sx={{
            flex: 1,
            backgroundColor: "#2D2D2D",
            p: 2,
            overflow: "auto",
            "&::-webkit-scrollbar": {
              width: "8px",
            },
            "&::-webkit-scrollbar-track": {
              background: "#1E1E1E",
              borderRadius: "4px",
            },
            "&::-webkit-scrollbar-thumb": {
              background: "#424242",
              borderRadius: "4px",
            },
          }}
        >
          <GameInfo gameState={gameState} onItemClick={handleItemClick} />
        </Paper>
      </Box>
    </Box>
  );
}
