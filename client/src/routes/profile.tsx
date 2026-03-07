import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Divider,
  Alert,
  CircularProgress,
} from "@mui/material";
import { useState } from "react";
import { isAuthenticated, getUserEmail, changePassword, signOut } from "@/services/auth.service";

export const Route = createFileRoute("/profile")({
  component: ProfilePage,
  beforeLoad: () => {
    if (!isAuthenticated()) {
      throw redirect({ to: "/login", search: { redirect: "/profile" } });
    }
  },
});

function ProfilePage() {
  const navigate = useNavigate();
  const userEmail = getUserEmail();

  // Change password form
  const [currentPw, setCurrentPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [confirmPw, setConfirmPw] = useState("");
  const [pwLoading, setPwLoading] = useState(false);
  const [pwSuccess, setPwSuccess] = useState(false);
  const [pwError, setPwError] = useState<string | null>(null);

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwError(null);
    setPwSuccess(false);

    if (newPw !== confirmPw) {
      setPwError("New passwords do not match.");
      return;
    }
    if (newPw.length < 8) {
      setPwError("New password must be at least 8 characters.");
      return;
    }

    setPwLoading(true);
    const result = await changePassword(currentPw, newPw);
    setPwLoading(false);

    if (result.success) {
      setPwSuccess(true);
      setCurrentPw("");
      setNewPw("");
      setConfirmPw("");
    } else {
      setPwError(result.error ?? "Failed to change password.");
    }
  };

  const handleSignOut = async () => {
    await signOut();
    navigate({ to: "/login" });
  };

  const SectionHeader = ({ children }: { children: React.ReactNode }) => (
    <Typography
      variant="h6"
      sx={{
        mb: 2,
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
    <Box
      sx={{
        display: "flex",
        justifyContent: "center",
        p: 4,
        minHeight: "calc(100vh - 64px)",
      }}
    >
      <Box sx={{ maxWidth: 560, width: "100%", display: "flex", flexDirection: "column", gap: 3 }}>
        <Typography
          variant="h3"
          sx={{
            textAlign: "center",
            textTransform: "uppercase",
            letterSpacing: "0.1em",
            fontSize: "2rem",
            borderBottom: "3px solid",
            borderColor: "primary.main",
            pb: 2,
          }}
        >
          Your Account
        </Typography>

        {/* Account info */}
        <Paper sx={{ p: 3 }}>
          <SectionHeader>Account Details</SectionHeader>
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            Email: <strong>{userEmail || "—"}</strong>
          </Typography>
        </Paper>

        {/* Change password */}
        <Paper sx={{ p: 3 }}>
          <SectionHeader>Change Passphrase</SectionHeader>
          <Box component="form" onSubmit={handleChangePassword} sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
            <TextField
              label="Current Passphrase"
              type="password"
              value={currentPw}
              onChange={(e) => setCurrentPw(e.target.value)}
              required
              fullWidth
              autoComplete="current-password"
            />
            <TextField
              label="New Passphrase"
              type="password"
              value={newPw}
              onChange={(e) => setNewPw(e.target.value)}
              required
              fullWidth
              autoComplete="new-password"
              helperText="Minimum 8 characters"
            />
            <TextField
              label="Confirm New Passphrase"
              type="password"
              value={confirmPw}
              onChange={(e) => setConfirmPw(e.target.value)}
              required
              fullWidth
              autoComplete="new-password"
              error={confirmPw.length > 0 && confirmPw !== newPw}
              helperText={confirmPw.length > 0 && confirmPw !== newPw ? "Passwords do not match" : ""}
            />
            {pwError && <Alert severity="error">{pwError}</Alert>}
            {pwSuccess && <Alert severity="success">Passphrase changed successfully.</Alert>}
            <Button
              type="submit"
              variant="contained"
              disabled={pwLoading}
              startIcon={pwLoading ? <CircularProgress size={18} color="inherit" /> : null}
            >
              {pwLoading ? "Changing..." : "Change Passphrase"}
            </Button>
          </Box>
        </Paper>

        {/* Danger zone */}
        <Paper sx={{ p: 3 }}>
          <SectionHeader>Session</SectionHeader>
          <Divider sx={{ mb: 2 }} />
          <Button
            variant="outlined"
            color="error"
            onClick={handleSignOut}
            fullWidth
          >
            Sign Out
          </Button>
        </Paper>
      </Box>
    </Box>
  );
}
