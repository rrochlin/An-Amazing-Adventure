import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import {
  TextField,
  Button,
  Box,
  Paper,
  Typography,
  Alert,
  Divider,
} from "@mui/material";
import { useState } from "react";
import { login } from "../services/auth.service";

export const Route = createFileRoute("/login")({
  component: RouteComponent,
});

function RouteComponent() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setIsLoading(true);
    const result = await login(email, password);
    setIsLoading(false);
    if (!result.success) {
      setError(result.error ?? "Incorrect credentials");
      return;
    }
    navigate({ to: "/" });
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
          Enter the Dungeon
        </Typography>
        <Typography
          variant="body2"
          align="center"
          sx={{ color: "text.secondary", mb: 2, fontStyle: "italic" }}
        >
          Speak your name and passphrase, adventurer.
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

        <form onSubmit={handleSubmit}>
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
            autoComplete="current-password"
          />

          {/* Forgot password — right-aligned under the password field */}
          <Box sx={{ textAlign: "right", mt: 0.5, mb: 1 }}>
            <Link
              to="/forgot-password"
              style={{ fontSize: "0.95rem", color: "inherit", opacity: 0.65, textDecoration: "underline" }}
            >
              Forgot your passphrase?
            </Link>
          </Box>

          <Button
            type="submit"
            fullWidth
            variant="contained"
            size="large"
            sx={{ mt: 2, mb: 1 }}
            disabled={isLoading}
          >
            {isLoading ? "Entering..." : "Enter"}
          </Button>
        </form>

        <Box sx={{ display: "flex", alignItems: "center", my: 2, gap: 1 }}>
          <Divider sx={{ flex: 1, borderColor: "primary.dark" }} />
          <Typography sx={{ color: "primary.main", fontSize: "1rem", opacity: 0.7, px: 1 }}>✦</Typography>
          <Divider sx={{ flex: 1, borderColor: "primary.dark" }} />
        </Box>

        <Typography variant="body2" align="center" sx={{ color: "text.secondary" }}>
          No account yet?{" "}
          <Link
            to="/signup"
            style={{ color: "inherit", fontWeight: 600, textDecoration: "underline" }}
          >
            Begin your adventure
          </Link>
        </Typography>
      </Paper>
    </Box>
  );
}
