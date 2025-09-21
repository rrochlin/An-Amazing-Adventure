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
      localStorage.setItem(
        `chatHistory-${sessionUUID}`,
        JSON.stringify(newChat),
      );
      return newChat;
    });
  };

  useEffect(() => {
    if (!gameState) return;
    localStorage.setItem(`gameState-${sessionUUID}`, JSON.stringify(gameState));
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

      // Get initial game state
      let state: GameState;
      const localState = localStorage.getItem(`gameState-${sessionUUID}`);
      if (localState != null && localState != "null") {
        state = JSON.parse(localState);
      } else {
        const gameResponse = await DescribeGame(sessionUUID);
        state = gameResponse.game_state;
      }
      setGameState(state);
      console.log(state);

      // Get initial narrative
      const previousChat = localStorage.getItem(`chatHistory-${sessionUUID}`);
      if (previousChat) {
        console.log(previousChat);
        setChatHistory(JSON.parse(previousChat));
      }

      //TODO probably should only do this now if it's actually needed
      if (!previousChat) {
        console.log("narrative not found, generating new narrative");
        const narrativeResponse = await SendChat(sessionUUID, {
          chat: "Please provide an introductory narrative for the player.",
        });
        handleSetChatHistory({
          type: "narrative",
          content: narrativeResponse.Response,
        });
        setGameState(narrativeResponse.game_state);
      }
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

      if (chat.Response) {
        handleSetChatHistory({
          type: "narrative",
          content: chat.Response,
        });
      } else {
        console.error("Invalid response format:", chat);
        setError("Received invalid response from server");
      }

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
      }}
    >
      <Box sx={{ flex: "0", minWidth: "20vw", p: 2 }}>
        <Paper sx={{ p: 2, backgroundColor: "#2D2D2D" }}>
          <Box
            sx={{
              height: "500px",
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
            }}
          >
            <Paper
              sx={{
                width: "18vw",
              }}
            >
              <RoomMap gameState={gameState} />
            </Paper>
          </Box>
          <GameInfo gameState={gameState} onItemClick={handleItemClick} />
        </Paper>
      </Box>

      <Box sx={{ flex: "1", p: 4, minHeight: 0 }}>
        <Paper sx={{ height: "100%", backgroundColor: "#2D2D2D" }}>
          <Chat
            chatHistory={chatHistory}
            command={command}
            setCommand={setCommand}
            handleCommand={handleCommand}
            isLoading={isLoading}
          />
        </Paper>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mt: 2, mx: 2 }}>
          {error}
        </Alert>
      )}
    </Box>
  );
}
