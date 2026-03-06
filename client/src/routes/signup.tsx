import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { TextField, Button, Box, Paper, Typography, Alert } from "@mui/material";
import { useState } from "react";
import { signUp, confirmSignUp } from "../services/auth.service";

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
      setError(result.error ?? "Confirmation failed");
      return;
    }
    navigate({ to: "/login" });
  };

  return (
    <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "50vh" }}>
      <Paper sx={{ p: 4, maxWidth: 400, width: "100%" }}>
        <Typography variant="h4" component="h1" gutterBottom align="center">
          {stage === "signup" ? "Sign Up" : "Confirm Email"}
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {stage === "signup" ? (
          <form onSubmit={handleSignUp}>
            <TextField fullWidth placeholder="Email" type="email" value={email}
              onChange={(e) => setEmail(e.target.value)} margin="normal" required autoComplete="email" />
            <TextField fullWidth placeholder="Password" type="password" value={password}
              onChange={(e) => setPassword(e.target.value)} margin="normal" required autoComplete="new-password" />
            <TextField fullWidth placeholder="Confirm Password" type="password" value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)} margin="normal" required autoComplete="new-password" />
            <Button type="submit" fullWidth variant="contained" sx={{ mt: 3, mb: 2 }} disabled={isLoading}>
              {isLoading ? "Creating Account..." : "Sign Up"}
            </Button>
          </form>
        ) : (
          <form onSubmit={handleConfirm}>
            <Typography variant="body2" sx={{ mb: 2 }}>
              A verification code has been sent to {email}. Enter it below.
            </Typography>
            <TextField fullWidth placeholder="Verification Code" value={confirmationCode}
              onChange={(e) => setConfirmationCode(e.target.value)} margin="normal" required />
            <Button type="submit" fullWidth variant="contained" sx={{ mt: 3, mb: 2 }} disabled={isLoading}>
              {isLoading ? "Confirming..." : "Confirm"}
            </Button>
          </form>
        )}

        <Typography variant="body2" align="center">
          Already have an account?{" "}
          <Link to="/login" style={{ color: "#1976d2", textDecoration: "none" }}>
            Sign In
          </Link>
        </Typography>
      </Paper>
    </Box>
  );
}
