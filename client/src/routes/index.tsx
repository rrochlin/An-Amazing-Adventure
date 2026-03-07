import { createFileRoute, redirect, useNavigate, useRouter } from "@tanstack/react-router";
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
  Alert,
  IconButton,
  Tooltip,
  CircularProgress,
} from "@mui/material";
import z from "zod";
import { isAuthenticated } from "@/services/auth.service";
import { ListGames, CreateGame, DeleteGame, type GameListItem } from "@/services/api.game";
import AddIcon from "@mui/icons-material/Add";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import DeleteIcon from "@mui/icons-material/Delete";
import { useState } from "react";

function GamesErrorFallback() {
  const router = useRouter();
  return (
    <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "calc(100vh - 64px)", p: 4 }}>
      <Paper sx={{ maxWidth: 480, width: "100%", p: 4, textAlign: "center" }}>
        <Typography variant="h5" sx={{ mb: 2, fontFamily: '"Cinzel", serif' }}>
          Could Not Load Adventures
        </Typography>
        <Alert severity="error" sx={{ mb: 3, textAlign: "left" }}>
          The server encountered an error. Your adventures are safe — please try again.
        </Alert>
        <Button variant="contained" onClick={() => router.invalidate()}>
          Retry
        </Button>
      </Paper>
    </Box>
  );
}

export const Route = createFileRoute("/")({
  validateSearch: z.object({
    count: z.number().optional(),
  }),
  component: RouteComponent,
  errorComponent: GamesErrorFallback,
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
  const [games, setGames] = useState<GameListItem[]>(
    Route.useLoaderData() as GameListItem[],
  );
  const [open, setOpen] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const navigate = useNavigate();
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;

  const handleClickOpen = () => {
    setCreateError(null);
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
    setCreateError(null);
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const formJson = Object.fromEntries((formData as any).entries());
    const name = formJson.characterName;
    try {
      const response = await CreateGame(name);
      handleClose();
      setGames((prev) => [
        ...prev,
        { player_name: name, session_id: response.session_id, ready: false },
      ]);
    } catch {
      setCreateError("Failed to create adventure — please try again.");
    }
  };

  const handleDelete = async (e: React.MouseEvent, sessionId: string) => {
    e.stopPropagation();
    setDeletingId(sessionId);
    try {
      await DeleteGame(sessionId);
      setGames((prev) => prev.filter((g) => g.session_id !== sessionId));
    } catch {
      // Non-fatal — show nothing, game remains in list
    } finally {
      setDeletingId(null);
    }
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
        {createError && (
          <Alert severity="error" sx={{ mx: 3, mb: 1 }}>{createError}</Alert>
        )}
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
            {games.map((game: GameListItem) => (
              <Paper
                key={game.session_id}
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
                    params: { sessionUUID: game.session_id },
                  })
                }
              >
                <Box sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                  <PlayArrowIcon sx={{ color: "primary.main", fontSize: "2rem" }} />
                  <Typography
                    variant="h6"
                    sx={{ fontSize: "1.5rem", fontFamily: '"Crimson Text", "Georgia", serif', flex: 1 }}
                  >
                    {game.player_name}
                  </Typography>
                  <Tooltip title="Delete adventure" placement="left">
                    <IconButton
                      size="small"
                      onClick={(e) => handleDelete(e, game.session_id)}
                      disabled={deletingId === game.session_id}
                      sx={{ color: "error.main", opacity: 0.6, "&:hover": { opacity: 1 } }}
                    >
                      {deletingId === game.session_id
                        ? <CircularProgress size={18} color="error" />
                        : <DeleteIcon fontSize="small" />}
                    </IconButton>
                  </Tooltip>
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
