import { createFileRoute, redirect, useNavigate, useRouter } from "@tanstack/react-router";
import {
  Button,
  Box,
  LinearProgress,
  Paper,
  Typography,
  useColorScheme,
  Divider,
  Alert,
  IconButton,
  Tooltip,
  CircularProgress,
  Chip,
} from "@mui/material";
import z from "zod";
import { isAuthenticated } from "@/services/auth.service";
import {
  ListGames,
  DeleteGame,
  type GameListItem,
  type ListGamesResponse,
  type UserQuotaInfo,
} from "@/services/api.game";
import AddIcon from "@mui/icons-material/Add";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import DeleteIcon from "@mui/icons-material/Delete";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
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
  loader: async (): Promise<ListGamesResponse> => {
    return await ListGames();
  },
});

function RouteComponent() {
  const data = Route.useLoaderData() as ListGamesResponse;
  const [games, setGames] = useState<GameListItem[]>(data.games ?? []);
  const [quota] = useState<UserQuotaInfo>(
    data.user_quota ?? { tokens_used: 0, token_limit: 0, ai_enabled: true, role: "user" },
  );
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const navigate = useNavigate();
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;

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

        {/* Token quota bar — only shown when AI is enabled and a limit is set */}
        {quota.ai_enabled && quota.token_limit > 0 && (
          <Box sx={{ mb: 2 }}>
            <Box sx={{ display: "flex", justifyContent: "space-between", mb: 0.5 }}>
              <Typography variant="caption" color="text.secondary">
                Token usage
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {quota.tokens_used.toLocaleString()} / {quota.token_limit.toLocaleString()}
              </Typography>
            </Box>
            <LinearProgress
              variant="determinate"
              value={Math.min(100, (quota.tokens_used / quota.token_limit) * 100)}
              color={
                quota.tokens_used / quota.token_limit > 0.9
                  ? "error"
                  : quota.tokens_used / quota.token_limit > 0.7
                    ? "warning"
                    : "primary"
              }
            />
          </Box>
        )}

        {/* Restricted notice — no AI access */}
        {!quota.ai_enabled && (
          <Alert severity="info" sx={{ mb: 2 }}>
            <strong>Preview Mode</strong> — AI narration is not yet enabled for your account.
            Contact the admin to request access.
          </Alert>
        )}

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
              <Box sx={{ display: "flex", alignItems: "flex-start", gap: 2 }}>
                <PlayArrowIcon sx={{ color: "primary.main", fontSize: "2rem", mt: 0.5, flexShrink: 0 }} />
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  {game.title ? (
                    <>
                      <Typography
                        variant="h6"
                        sx={{ fontSize: "1.4rem", fontFamily: '"Cinzel", "Georgia", serif', lineHeight: 1.2 }}
                      >
                        {game.title}
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          color: "text.secondary",
                          fontFamily: '"Crimson Text", "Georgia", serif',
                          fontSize: "1rem",
                          mt: 0.25,
                        }}
                      >
                        Playing as {game.player_name}
                      </Typography>
                    </>
                  ) : (
                    <Typography
                      variant="h6"
                      sx={{ fontSize: "1.5rem", fontFamily: '"Crimson Text", "Georgia", serif' }}
                    >
                      {game.player_name}
                    </Typography>
                  )}
                  {!game.ready && (
                    <Chip
                      label="Generating..."
                      size="small"
                      color="secondary"
                      sx={{ mt: 0.75, fontSize: "0.75rem" }}
                    />
                  )}
                </Box>
                <Box sx={{ display: "flex", gap: 0.5, flexShrink: 0 }}>
                  <Tooltip title="Adventure details" placement="left">
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate({
                          to: "/game-{$sessionUUID}/details",
                          params: { sessionUUID: game.session_id },
                        });
                      }}
                      sx={{ color: "primary.main", opacity: 0.6, "&:hover": { opacity: 1 } }}
                    >
                      <InfoOutlinedIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
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
            onClick={() => navigate({ to: "/create" })}
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
  );
}
