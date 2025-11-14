import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Box,
  Paper,
  Typography,
  TextField,
  useColorScheme,
  Divider,
} from "@mui/material";
import z from "zod";
import { isAuthenticated } from "@/services/auth.service";
import { ListGames, StartGame } from "@/services/api.game";
import AddIcon from "@mui/icons-material/Add";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import { useState } from "react";
import type { ListGamesResponse } from "@/types/api.types";

export const Route = createFileRoute("/")({
  validateSearch: z.object({
    count: z.number().optional(),
  }),
  component: RouteComponent,
  beforeLoad: () => {
    if (!isAuthenticated()) {
      throw redirect({
        to: "/login",
        search: { redirect: location.href },
      });
    }
  },
  loader: async () => {
    const games = await ListGames();
    return games;
  },
});

function RouteComponent() {
  const [games, setGames] = useState<ListGamesResponse[]>(
    Route.useLoaderData(),
  );
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;

  const handleClickOpen = () => {
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const formJson = Object.fromEntries((formData as any).entries());
    const name = formJson.characterName;
    const response = await StartGame({ playerName: name });
    handleClose();
    if (!response.success) {
      console.error(response.error);
      alert("Error creating game please try again later");
      return;
    }
    setGames((prev) => [
      ...prev,
      { playerName: name, sessionId: response.sessionUUID },
    ]);
  };

  return (
    <>
      <Dialog
        open={open}
        onClose={handleClose}
        PaperProps={{
          sx: {
            backgroundImage: 'linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))',
            border: '1px solid rgba(201, 169, 98, 0.2)',
          }
        }}
      >
        <DialogTitle sx={{
          fontFamily: '"Cinzel", "Georgia", serif',
          fontSize: "1.75rem",
          textAlign: "center",
          borderBottom: "2px solid",
          borderColor: "primary.main",
        }}>
          Begin Your Adventure
        </DialogTitle>
        <DialogContent>
          <DialogContentText sx={{
            mt: 2,
            mb: 2,
            fontSize: "1.1rem",
            fontFamily: '"Crimson Text", "Georgia", serif',
          }}>
            Enter a name for your adventurer to begin questing!
          </DialogContentText>
          <form onSubmit={handleSubmit} id="game-form">
            <TextField
              autoFocus
              required
              margin="dense"
              id="name"
              name="characterName"
              label="Character Name"
              type="text"
              fullWidth
              variant="outlined"
              sx={{
                "& .MuiInputBase-input": {
                  fontSize: "1.1rem",
                  fontFamily: '"Crimson Text", "Georgia", serif',
                },
                "& .MuiInputLabel-root": {
                  fontSize: "1.1rem",
                  fontFamily: '"Crimson Text", "Georgia", serif',
                }
              }}
            />
          </form>
        </DialogContent>
        <DialogActions sx={{ p: 2, gap: 1 }}>
          <Button onClick={handleClose} variant="outlined" sx={{ fontSize: "1rem" }}>
            Cancel
          </Button>
          <Button type="submit" form="game-form" variant="contained" sx={{ fontSize: "1rem" }}>
            Create World
          </Button>
        </DialogActions>
      </Dialog>

      <Box sx={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "calc(100vh - 64px)",
        p: 4,
      }}>
        <Paper
          sx={{
            maxWidth: 800,
            width: "100%",
            p: 4,
            transition: "all 0.3s ease-in-out",
            "&:hover": {
              boxShadow: "0 6px 24px rgba(0, 0, 0, 0.6), inset 0 1px 0 rgba(201, 169, 98, 0.2)",
            }
          }}
        >
          <Typography
            variant="h3"
            sx={{
              mb: 3,
              textAlign: "center",
              textTransform: "uppercase",
              letterSpacing: "0.1em",
              fontSize: "2.5rem",
              borderBottom: "3px solid",
              borderColor: "primary.main",
              pb: 2,
            }}
          >
            Your Adventures
          </Typography>

          <Typography
            variant="body1"
            sx={{
              mb: 3,
              textAlign: "center",
              fontStyle: "italic",
              fontSize: "1.2rem",
              color: "text.secondary",
            }}
          >
            Select a character to continue your quest, or forge a new path
          </Typography>

          <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
            {games.map((game: ListGamesResponse) => (
              <Paper
                key={game.sessionId}
                sx={{
                  p: 3,
                  cursor: "pointer",
                  transition: "all 0.2s ease-in-out",
                  backgroundColor: isDark
                    ? "rgba(201, 169, 98, 0.05)"
                    : "rgba(160, 130, 109, 0.15)",
                  border: isDark
                    ? "1px solid rgba(201, 169, 98, 0.2)"
                    : "2px solid rgba(139, 111, 71, 0.4)",
                  "&:hover": {
                    backgroundColor: isDark
                      ? "rgba(201, 169, 98, 0.1)"
                      : "rgba(160, 130, 109, 0.25)",
                    transform: "translateY(-2px)",
                    boxShadow: "0 4px 12px rgba(201, 169, 98, 0.3)",
                  },
                }}
                onClick={() =>
                  navigate({
                    to: "/game-{$sessionUUID}",
                    params: { sessionUUID: game.sessionId },
                  })
                }
              >
                <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                  <PlayArrowIcon sx={{ color: "primary.main", fontSize: "2rem" }} />
                  <Typography
                    variant="h6"
                    sx={{
                      fontSize: "1.5rem",
                      fontFamily: '"Crimson Text", "Georgia", serif',
                    }}
                  >
                    {game.playerName}
                  </Typography>
                </Box>
              </Paper>
            ))}

            <Divider sx={{ my: 2 }} />

            <Paper
              sx={{
                p: 3,
                cursor: "pointer",
                transition: "all 0.2s ease-in-out",
                backgroundColor: isDark
                  ? "rgba(106, 78, 157, 0.1)"
                  : "rgba(139, 111, 71, 0.15)",
                border: isDark
                  ? "2px dashed rgba(106, 78, 157, 0.5)"
                  : "2px dashed rgba(139, 111, 71, 0.5)",
                "&:hover": {
                  backgroundColor: isDark
                    ? "rgba(106, 78, 157, 0.2)"
                    : "rgba(139, 111, 71, 0.25)",
                  transform: "translateY(-2px)",
                  boxShadow: "0 4px 12px rgba(106, 78, 157, 0.4)",
                },
              }}
              onClick={handleClickOpen}
            >
              <Box sx={{ display: "flex", alignItems: "center", gap: 2, justifyContent: "center" }}>
                <AddIcon sx={{ color: "secondary.main", fontSize: "2rem" }} />
                <Typography
                  variant="h6"
                  sx={{
                    fontSize: "1.5rem",
                    fontFamily: '"Crimson Text", "Georgia", serif',
                    color: "secondary.main",
                  }}
                >
                  Create a New Adventure
                </Typography>
              </Box>
            </Paper>
          </Box>
        </Paper>
      </Box>
    </>
  );
}
