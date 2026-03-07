import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import {
  Box,
  Button,
  Chip,
  Paper,
  Step,
  StepLabel,
  Stepper,
  TextField,
  Typography,
  Alert,
} from "@mui/material";
import { useState } from "react";
import { isAuthenticated } from "@/services/auth.service";
import { CreateGame } from "@/services/api.game";

const PREFERENCE_OPTIONS = [
  { label: "Combat", value: "combat" },
  { label: "Puzzles", value: "puzzles" },
  { label: "Dialog", value: "dialog" },
  { label: "Exploration", value: "exploration" },
  { label: "Chance", value: "chance" },
  { label: "Stealth", value: "stealth" },
  { label: "Crafting", value: "crafting" },
  { label: "Mystery", value: "mystery" },
];

const STEPS = ["Your Character", "Adventure Preferences"];

export const Route = createFileRoute("/create")({
  component: CreateRoute,
  beforeLoad: () => {
    if (!isAuthenticated()) {
      throw redirect({ to: "/login", search: { redirect: location.href } });
    }
  },
});

function CreateRoute() {
  const navigate = useNavigate();
  const [step, setStep] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Character fields
  const [playerName, setPlayerName] = useState("");
  const [playerAge, setPlayerAge] = useState("");
  const [playerDescription, setPlayerDescription] = useState("");
  const [playerBackstory, setPlayerBackstory] = useState("");

  // Adventure preference fields
  const [preferences, setPreferences] = useState<string[]>([]);
  const [themeHint, setThemeHint] = useState("");

  const togglePreference = (value: string) => {
    setPreferences((prev) =>
      prev.includes(value) ? prev.filter((p) => p !== value) : [...prev, value],
    );
  };

  const handleNext = () => setStep((s) => s + 1);
  const handleBack = () => setStep((s) => s - 1);

  const handleSubmit = async () => {
    setError(null);
    setIsSubmitting(true);
    try {
      const result = await CreateGame({
        player_name: playerName.trim() || undefined,
        player_age: playerAge.trim() || undefined,
        player_description: playerDescription.trim() || undefined,
        player_backstory: playerBackstory.trim() || undefined,
        theme_hint: themeHint.trim() || undefined,
        preferences: preferences.length > 0 ? preferences : undefined,
      });
      navigate({
        to: "/game-{$sessionUUID}",
        params: { sessionUUID: result.session_id },
      });
    } catch {
      setError("Failed to create adventure — please try again.");
      setIsSubmitting(false);
    }
  };

  const allBlank =
    !playerName.trim() &&
    !playerAge.trim() &&
    !playerDescription.trim() &&
    !playerBackstory.trim() &&
    preferences.length === 0 &&
    !themeHint.trim();

  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: "center",
        alignItems: "flex-start",
        minHeight: "calc(100vh - 64px)",
        p: 4,
        pt: 6,
      }}
    >
      <Paper
        sx={{
          maxWidth: 680,
          width: "100%",
          p: 4,
          backgroundImage:
            "linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))",
          border: "1px solid rgba(201, 169, 98, 0.2)",
        }}
      >
        <Typography
          variant="h3"
          sx={{
            mb: 1,
            textAlign: "center",
            textTransform: "uppercase",
            letterSpacing: "0.1em",
            fontSize: "2rem",
            borderBottom: "3px solid",
            borderColor: "primary.main",
            pb: 2,
          }}
        >
          Forge Your Adventure
        </Typography>

        <Stepper activeStep={step} sx={{ mb: 4, mt: 3 }}>
          {STEPS.map((label) => (
            <Step key={label}>
              <StepLabel>{label}</StepLabel>
            </Step>
          ))}
        </Stepper>

        {step === 0 && (
          <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography
              variant="body2"
              sx={{
                color: "text.secondary",
                fontStyle: "italic",
                fontFamily: '"Crimson Text", "Georgia", serif',
                fontSize: "1rem",
              }}
            >
              All fields are optional — leave anything blank and the AI will
              craft it for you.
            </Typography>

            <TextField
              label="Character Name"
              value={playerName}
              onChange={(e) => setPlayerName(e.target.value)}
              fullWidth
              helperText="What are you called, adventurer?"
              inputProps={{ maxLength: 60 }}
            />

            <TextField
              label="Age"
              value={playerAge}
              onChange={(e) => setPlayerAge(e.target.value)}
              fullWidth
              helperText='e.g. "early 30s", "elderly", "young"'
              inputProps={{ maxLength: 40 }}
            />

            <TextField
              label="Description"
              value={playerDescription}
              onChange={(e) => setPlayerDescription(e.target.value)}
              fullWidth
              multiline
              minRows={3}
              helperText="Physical appearance, personality, skills — anything that defines your character"
              inputProps={{ maxLength: 500 }}
            />

            <TextField
              label="Backstory"
              value={playerBackstory}
              onChange={(e) => setPlayerBackstory(e.target.value)}
              fullWidth
              multiline
              minRows={4}
              helperText="Your character's history, motivations, and secrets"
              inputProps={{ maxLength: 1000 }}
            />

            <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
              <Button
                variant="outlined"
                onClick={() => navigate({ to: "/" })}
              >
                Cancel
              </Button>
              <Button variant="contained" onClick={handleNext}>
                Next: Adventure Preferences
              </Button>
            </Box>
          </Box>
        )}

        {step === 1 && (
          <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography
              variant="body2"
              sx={{
                color: "text.secondary",
                fontStyle: "italic",
                fontFamily: '"Crimson Text", "Georgia", serif',
                fontSize: "1rem",
              }}
            >
              Shape the world you&apos;ll explore. Select as many or as few as
              you like — or skip everything and let the AI surprise you.
            </Typography>

            <Box>
              <Typography
                variant="subtitle2"
                sx={{ mb: 1.5, textTransform: "uppercase", letterSpacing: "0.08em" }}
              >
                Preferred Gameplay
              </Typography>
              <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                {PREFERENCE_OPTIONS.map((opt) => (
                  <Chip
                    key={opt.value}
                    label={opt.label}
                    clickable
                    onClick={() => togglePreference(opt.value)}
                    color={preferences.includes(opt.value) ? "primary" : "default"}
                    variant={preferences.includes(opt.value) ? "filled" : "outlined"}
                    sx={{ fontSize: "0.9rem", py: 0.5 }}
                  />
                ))}
              </Box>
            </Box>

            <TextField
              label="World Tone / Theme Hint"
              value={themeHint}
              onChange={(e) => setThemeHint(e.target.value)}
              fullWidth
              helperText='e.g. "gritty noir", "high fantasy epic", "light-hearted comedy", "cosmic horror"'
              inputProps={{ maxLength: 200 }}
            />

            {allBlank && (
              <Alert severity="info" sx={{ mt: 1 }}>
                No details provided — the AI will generate your entire character
                and world from scratch.
              </Alert>
            )}

            {error && (
              <Alert severity="error">{error}</Alert>
            )}

            <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
              <Button variant="outlined" onClick={handleBack} disabled={isSubmitting}>
                Back
              </Button>
              <Button
                variant="contained"
                onClick={handleSubmit}
                disabled={isSubmitting}
                sx={{ minWidth: 160 }}
              >
                {isSubmitting ? "Creating..." : "Begin Adventure"}
              </Button>
            </Box>
          </Box>
        )}
      </Paper>
    </Box>
  );
}
