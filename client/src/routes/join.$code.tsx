import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  Typography,
  Chip,
  LinearProgress,
} from "@mui/material";
import { isAuthenticated } from "../services/auth.service";
import { GetInvite, JoinInvite } from "../services/api.invites";
import type { InviteInfo } from "../types/types";

export const Route = createFileRoute("/join/$code")({
  component: JoinPage,
});

function JoinPage() {
  const { code } = Route.useParams();
  const navigate = useNavigate();
  const authed = isAuthenticated();

  const [invite, setInvite] = useState<InviteInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [joining, setJoining] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    GetInvite(code)
      .then((info) => setInvite(info))
      .catch(() => setError("Invite not found or expired"))
      .finally(() => setLoading(false));
  }, [code]);

  const handleJoin = async () => {
    if (!authed) {
      sessionStorage.setItem("pendingInviteCode", code);
      navigate({ to: "/login" });
      return;
    }
    setJoining(true);
    setError(null);
    try {
      const res = await JoinInvite(code);
      navigate({ to: "/game-{$sessionUUID}", params: { sessionUUID: res.session_id } });
    } catch {
      setError("Failed to join — the invite may have expired or the party is full");
    } finally {
      setJoining(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "60vh" }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error && !invite) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "calc(100vh - 64px)",
        p: 4,
      }}
    >
      <Paper sx={{ maxWidth: 440, width: "100%", p: 4 }}>
        <Typography
          variant="h5"
          sx={{ mb: 1, fontFamily: '"Cinzel", serif', textAlign: "center" }}
        >
          Party Invitation
        </Typography>

        {invite && (
          <>
            <Typography
              variant="h6"
              sx={{ mb: 2, textAlign: "center", color: "primary.main" }}
            >
              {invite.game_title || "An Adventure Awaits"}
            </Typography>

            <Box sx={{ mb: 2 }}>
              <Box sx={{ display: "flex", justifyContent: "space-between", mb: 0.5 }}>
                <Typography variant="body2" sx={{ color: "text.secondary" }}>
                  Party
                </Typography>
                <Typography variant="body2">
                  {invite.party_current} / {invite.party_max} members
                </Typography>
              </Box>
              <LinearProgress
                variant="determinate"
                value={(invite.party_current / invite.party_max) * 100}
                sx={{ height: 6, borderRadius: 3 }}
              />
            </Box>

            {invite.expired && (
              <Chip
                label="Expired or Full"
                color="error"
                size="small"
                sx={{ mb: 2, display: "block", textAlign: "center" }}
              />
            )}

            {error && (
              <Alert severity="error" sx={{ mb: 2 }}>
                {error}
              </Alert>
            )}

            {!authed && (
              <Alert severity="info" sx={{ mb: 2 }}>
                You need to sign in to join this adventure.
              </Alert>
            )}

            <Button
              variant="contained"
              fullWidth
              disabled={invite.expired || joining}
              onClick={handleJoin}
              sx={{ mt: 1 }}
            >
              {joining
                ? "Joining..."
                : authed
                ? `Join "${invite.game_title || "Adventure"}"`
                : "Sign in to Join"}
            </Button>

            {!authed && (
              <Button
                variant="text"
                fullWidth
                sx={{ mt: 1 }}
                onClick={() => {
                  sessionStorage.setItem("pendingInviteCode", code);
                  navigate({ to: "/signup" });
                }}
              >
                Create account
              </Button>
            )}
          </>
        )}
      </Paper>
    </Box>
  );
}
