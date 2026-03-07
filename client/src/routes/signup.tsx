import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import {
  TextField,
  Button,
  Box,
  Paper,
  Typography,
  Alert,
  Divider,
  InputAdornment,
} from "@mui/material";
import { useState } from "react";
import { signUp, confirmSignUp, resendConfirmationCode } from "../services/auth.service";

export const Route = createFileRoute("/signup")({
  component: RouteComponent,
});

function RouteComponent() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [confirmationCode, setConfirmationCode] = useState("");
  const [stage, setStage] = useState<"signup" | "confirm">("signup");
  const [error, setError] = useState("");
  const [resendStatus, setResendStatus] = useState<"idle" | "sending" | "sent">("idle");
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();

  const handleSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    setIsLoading(true);
    const result = await signUp(email, password);
    setIsLoading(false);
    if (!result.success) {
      setError(result.error ?? "Sign up failed");
      return;
    }
    setStage("confirm");
  };

  const handleConfirm = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setIsLoading(true);
    const result = await confirmSignUp(email, confirmationCode);
    setIsLoading(false);
    if (!result.success) {
      setError(result.error ?? "Confirmation failed — the seal may be expired");
      return;
    }
    navigate({ to: "/login" });
  };

  const handleResend = async () => {
    setResendStatus("sending");
    await resendConfirmationCode(email);
    setResendStatus("sent");
    setTimeout(() => setResendStatus("idle"), 5000);
  };

  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "70vh",
        pt: 4,
      }}
    >
      <Paper
        sx={{
          p: { xs: 3, sm: 5 },
          maxWidth: 440,
          width: "100%",
          position: "relative",
          overflow: "hidden",
          "&::before": {
            content: '"⚔"',
            position: "absolute",
            top: 12,
            left: 16,
            fontSize: "1.1rem",
            opacity: 0.15,
            color: "primary.main",
            pointerEvents: "none",
          },
          "&::after": {
            content: '"⚔"',
            position: "absolute",
            top: 12,
            right: 16,
            fontSize: "1.1rem",
            opacity: 0.15,
            color: "primary.main",
            pointerEvents: "none",
            transform: "scaleX(-1)",
          },
        }}
      >
        <Typography
          variant="h4"
          component="h1"
          align="center"
          sx={{ mb: 0.5, color: "primary.main" }}
        >
          {stage === "signup" ? "Forge Your Legend" : "Prove Your Worth"}
        </Typography>
        <Typography
          variant="body2"
          align="center"
          sx={{ color: "text.secondary", mb: 2, fontStyle: "italic" }}
        >
          {stage === "signup"
            ? "Register your name in the annals of the dungeon."
            : `An enchanted seal was dispatched to ${email}. Enter it to complete your oath.`}
        </Typography>

        <Box sx={{ display: "flex", alignItems: "center", my: 2, gap: 1 }}>
          <Divider sx={{ flex: 1, borderColor: "primary.dark" }} />
          <Typography sx={{ color: "primary.main", fontSize: "1rem", opacity: 0.7, px: 1 }}>✦</Typography>
          <Divider sx={{ flex: 1, borderColor: "primary.dark" }} />
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {/* ── Stage: signup ── */}
        {stage === "signup" && (
          <form onSubmit={handleSignUp}>
            <TextField
              fullWidth
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              margin="normal"
              required
              autoComplete="email"
              autoFocus
            />
            <TextField
              fullWidth
              label="Password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              margin="normal"
              required
              autoComplete="new-password"
              helperText="At least 8 characters, uppercase, lowercase & number"
            />
            <TextField
              fullWidth
              label="Confirm Password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              margin="normal"
              required
              autoComplete="new-password"
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              size="large"
              sx={{ mt: 3, mb: 1 }}
              disabled={isLoading}
            >
              {isLoading ? "Registering..." : "Register"}
            </Button>
          </form>
        )}

        {/* ── Stage: confirm ── */}
        {stage === "confirm" && (
          <form onSubmit={handleConfirm}>
            <TextField
              fullWidth
              label="Verification seal"
              value={confirmationCode}
              onChange={(e) => setConfirmationCode(e.target.value.trim())}
              margin="normal"
              required
              autoFocus
              inputProps={{ maxLength: 8, inputMode: "numeric" }}
              helperText="6-digit code from the enchanted missive"
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <Typography sx={{ fontSize: "1rem", opacity: 0.5 }}>🔑</Typography>
                  </InputAdornment>
                ),
              }}
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              size="large"
              sx={{ mt: 3, mb: 1 }}
              disabled={isLoading}
            >
              {isLoading ? "Swearing the oath..." : "Swear the Oath"}
            </Button>

            {/* Resend code */}
            <Box sx={{ textAlign: "center", mt: 1 }}>
              {resendStatus === "sent" ? (
                <Typography variant="body2" sx={{ color: "success.main", fontStyle: "italic" }}>
                  A new scroll has been dispatched ✓
                </Typography>
              ) : (
                <Button
                  variant="text"
                  size="small"
                  onClick={handleResend}
                  disabled={resendStatus === "sending"}
                  sx={{ color: "text.secondary", textTransform: "none", fontSize: "0.95rem" }}
                >
                  {resendStatus === "sending" ? "Sending..." : "Didn't receive it? Send again"}
                </Button>
              )}
            </Box>
          </form>
        )}

        <Box sx={{ display: "flex", alignItems: "center", my: 2, gap: 1 }}>
          <Divider sx={{ flex: 1, borderColor: "primary.dark" }} />
          <Typography sx={{ color: "primary.main", fontSize: "1rem", opacity: 0.7, px: 1 }}>✦</Typography>
          <Divider sx={{ flex: 1, borderColor: "primary.dark" }} />
        </Box>

        <Typography variant="body2" align="center" sx={{ color: "text.secondary" }}>
          Already registered?{" "}
          <Link
            to="/login"
            style={{ color: "inherit", fontWeight: 600, textDecoration: "underline" }}
          >
            Enter the dungeon
          </Link>
        </Typography>
      </Paper>
    </Box>
  );
}
