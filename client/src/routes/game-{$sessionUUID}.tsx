import { createFileRoute, redirect } from "@tanstack/react-router"
import { useState } from "react";
import type { ChatMessageType, GameState } from '../types/types'
import { Chat as SendChat, DescribeGame, StartGame, WorldReady } from "../services/api.game";
import { Alert, Box, Button, CircularProgress, Paper, Typography } from "@mui/material";
import { RoomMap } from "../components/RoomMap";
import { GameInfo } from "../components/GameInfo";
import { Chat } from "../components/Chat";
import { isAuthenticated } from "../services/auth.service";

export const Route = createFileRoute('/game-{$sessionUUID}')({
	component: PostComponent,
	beforeLoad: async ({ location }) => {
		if (!isAuthenticated()) {
			throw redirect({
				to: '/login',
				search: { redirect: location.href },
			})
		}
	},
	loader: async ({ params }) => {
		return StartGame(params.sessionUUID)
	}
})

function PostComponent() {
	const { sessionUUID } = Route.useParams()
	const [gameState, setGameState] = useState<GameState | null>(null);
	const [command, setCommand] = useState("");
	const [gameStarted, setGameStarted] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [isLoading, setIsLoading] = useState(false);
	const [pollingStatus, setPollingStatus] = useState<string>("");
	const [chatHistory, setChatHistory] = useState<ChatMessageType[]>([]);

	const handleSetChatHistory = (message: ChatMessageType) => {
		setChatHistory((prevChatHistory) => {
			const newChat = [...prevChatHistory, message];
			localStorage.setItem("chatHistory", JSON.stringify(newChat));
			return newChat;
		});
	};

	const startGame = async () => {
		setIsLoading(true);
		setError(null);
		setPollingStatus("Starting game...");

		try {
			const response = await StartGame(sessionUUID)
			if (response.ready)
				setGameStarted(true);

			// Poll for world generation completion
			const worldGenerated = await WorldReady(sessionUUID)
			if (!worldGenerated) {
				setError(
					"World generation is taking longer than expected. You can still try to play, but some features might not be available yet.",
				);
			}

			// Get initial game state
			setPollingStatus("Loading initial game state...");
			const gameResponse = await DescribeGame(sessionUUID)
			setGameState(gameResponse.game_state);

			// Get initial narrative
			const previousChat = localStorage.getItem("chatHistory");
			if (previousChat) {
				console.log(previousChat);
				setChatHistory(JSON.parse(previousChat));
			}

			//TODO probably should only do this now if it's actually needed
			const narrativeResponse = await SendChat(sessionUUID, {
				chat: "Please provide an introductory narrative for the player.",
			});
			if (narrativeResponse.Response) {
				handleSetChatHistory({
					type: "narrative",
					content: narrativeResponse.Response,
				});
			}
			if (narrativeResponse.game_state) {
				setGameState((prev) => ({
					...prev!,
					current_room: narrativeResponse.game_state.current_room,
					inventory: narrativeResponse.game_state.inventory,
					rooms: {
						...prev!.rooms,
						...narrativeResponse.game_state.rooms,
					},
				}));
			} else {
				// Fallback to regular game state update
				const gameResponse = await DescribeGame(sessionUUID)
				setGameState(gameResponse.game_state);
			}
		} catch (err) {
			setError(
				"Failed to start game. Please check if the server is running and try again.",
			);
			console.error("Error starting game:", err);
		} finally {
			setIsLoading(false);
			setPollingStatus("");
		}
	};

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

			// Update game state from chat response if available
			if (chat.game_state) {
				setGameState((prev) => ({
					...prev!,
					current_room: chat.game_state.current_room,
					inventory: chat.game_state.inventory,
					rooms: {
						...prev!.rooms,
						...chat.game_state.rooms,
					},
				}));
			} else {
				// Fallback to regular game state update
				const gameResponse = await DescribeGame(sessionUUID)
				setGameState(gameResponse.game_state);
			}

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

	if (!gameStarted) {
		return (
			<Box sx={{ p: 4, textAlign: "center" }}>
				<Typography variant="h4" sx={{ mb: 4 }}>
					Text Adventure Game
				</Typography>
				<Button variant="contained" onClick={startGame} disabled={isLoading}>
					{isLoading ? <CircularProgress size={24} /> : "Start Game"}
				</Button>
				{pollingStatus && (
					<Typography sx={{ mt: 2 }}>{pollingStatus}</Typography>
				)}
				{error && (
					<Alert severity="error" sx={{ mt: 2 }}>
						{error}
					</Alert>
				)}
			</Box>
		);
	}

	if (!gameState) {
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
				height: "100vh",
				display: "flex",
				flexDirection: "row",
				overflow: "hidden",
				backgroundColor: "#1E1E1E",
			}}
		>
			<Box sx={{ flex: "0", width: "20vw", p: 2 }}>
				<Paper sx={{ p: 2, backgroundColor: "#2D2D2D" }}>
					<Box
						sx={{
							height: "500px",
							display: "flex",
							justifyContent: "center",
							alignItems: "center",
						}}
					>
						<RoomMap gameState={gameState} />
					</Box>
					<GameInfo gameState={gameState} onItemClick={handleItemClick} />
				</Paper>
			</Box>

			<Box sx={{ flex: "1 1 auto", p: 2, minHeight: 0 }}>
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
