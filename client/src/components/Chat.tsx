import { Box, Button, CircularProgress, Paper, TextField, useColorScheme } from "@mui/material";
import { useEffect, useRef } from "react";
import { type ChatMessageType } from "../types/types";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

const LoadingMessage = () => {
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;

  const flavorTexts = [
    "The Dungeon Master ponders...",
    "Rolling the dice of fate...",
    "Consulting ancient tomes...",
    "Weaving threads of destiny...",
    "The story unfolds...",
    "Channeling arcane energies...",
    "Shadows whisper secrets...",
    "The fates are deciding...",
    "Scrying into the unknown...",
    "Invoking the spirits...",
  ];

  // Pick a random flavor text
  const flavorText = useRef(
    flavorTexts[Math.floor(Math.random() * flavorTexts.length)]
  ).current;

  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: "flex-start",
        mb: 2,
        animation: "fadeIn 0.3s ease-in",
        "@keyframes fadeIn": {
          "0%": {
            opacity: 0,
            transform: "translateY(10px)",
          },
          "100%": {
            opacity: 1,
            transform: "translateY(0)",
          },
        },
      }}
    >
      <Paper
        sx={{
          p: 2,
          maxWidth: "300px",
          backgroundColor: isDark
            ? "rgba(201, 169, 98, 0.1)"
            : "rgba(160, 130, 109, 0.2)",
          border: isDark
            ? "1px solid rgba(201, 169, 98, 0.3)"
            : "2px solid #A0826D",
          borderRadius: 2,
          boxShadow: isDark
            ? "0 2px 8px rgba(0, 0, 0, 0.5)"
            : "0 2px 4px rgba(107, 86, 56, 0.3)",
        }}
      >
        <Box
          sx={{
            fontFamily: "Crimson Text, Georgia, serif",
            fontSize: "1.1rem",
            fontStyle: "italic",
            position: "relative",
            background: isDark
              ? "linear-gradient(90deg, #B8A588 0%, #FFD700 50%, #B8A588 100%)"
              : "linear-gradient(90deg, #5D4037 0%, #8B6F47 50%, #5D4037 100%)",
            backgroundSize: "200% 100%",
            backgroundClip: "text",
            WebkitBackgroundClip: "text",
            WebkitTextFillColor: "transparent",
            animation: "shimmer 2s linear infinite",
            textShadow: "none",
            "@keyframes shimmer": {
              "0%": {
                backgroundPosition: "100% 0",
              },
              "100%": {
                backgroundPosition: "-100% 0",
              },
            },
            // Add glowing effect that follows the shimmer
            "&::after": {
              content: '""',
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              background: isDark
                ? "linear-gradient(90deg, transparent 0%, rgba(255, 215, 0, 0.15) 50%, transparent 100%)"
                : "linear-gradient(90deg, transparent 0%, rgba(139, 111, 71, 0.2) 50%, transparent 100%)",
              backgroundSize: "200% 100%",
              animation: "shimmer 2s linear infinite",
              pointerEvents: "none",
              filter: "blur(3px)",
            },
          }}
        >
          {flavorText}
        </Box>
      </Paper>
    </Box>
  );
};

const ChatMessage = ({ message }: { message: ChatMessageType }) => {
  const isPlayer = message.type === "player";
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;

  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: isPlayer ? "flex-end" : "flex-start",
        mb: 2,
        animation: "fadeIn 0.3s ease-in",
        "@keyframes fadeIn": {
          "0%": {
            opacity: 0,
            transform: "translateY(10px)",
          },
          "100%": {
            opacity: 1,
            transform: "translateY(0)",
          },
        },
      }}
    >
      <Paper
        sx={{
          p: 2,
          maxWidth: "min(600px, 85%)",
          backgroundColor:
            isDark
              ? isPlayer
                ? "rgba(106, 78, 157, 0.3)"
                : "rgba(201, 169, 98, 0.1)"
              : isPlayer
              ? "rgba(139, 111, 71, 0.25)"
              : "rgba(160, 130, 109, 0.2)",
          border:
            isDark
              ? isPlayer
                ? "1px solid #6B4E9D"
                : "1px solid rgba(201, 169, 98, 0.3)"
              : isPlayer
              ? "2px solid #8B6F47"
              : "2px solid #A0826D",
          color: "text.primary",
          borderRadius: 2,
          boxShadow:
            isDark
              ? isPlayer
                ? "0 2px 8px rgba(106, 78, 157, 0.3)"
                : "0 2px 8px rgba(0, 0, 0, 0.5)"
              : "0 2px 4px rgba(107, 86, 56, 0.3)",
          wordWrap: "break-word",
          overflowWrap: "break-word",
          transition: "all 0.2s ease-in-out",
          "&:hover": {
            transform: "translateY(-2px)",
            boxShadow: isPlayer
              ? "0 4px 12px rgba(106, 78, 157, 0.4)"
              : "0 4px 12px rgba(201, 169, 98, 0.3)",
          },
          "& p": {
            margin: 0,
            marginBottom: "8px",
            "&:last-child": {
              marginBottom: 0,
            },
          },
          "& pre": {
            backgroundColor: "#1E1E1E",
            padding: "8px",
            borderRadius: "4px",
            overflow: "auto",
            maxWidth: "100%",
            "& code": {
              fontSize: "0.875rem",
              fontFamily: "monospace",
            },
          },
          "& code": {
            backgroundColor: "#1E1E1E",
            padding: "2px 6px",
            borderRadius: "3px",
            fontSize: "0.875rem",
            fontFamily: "monospace",
          },
        }}
      >
        <Markdown remarkPlugins={[remarkGfm]}>{message.content}</Markdown>
      </Paper>
    </Box>
  );
};

export const Chat = ({
  chatHistory,
  command,
  setCommand,
  handleCommand,
  isLoading,
}: {
  chatHistory: ChatMessageType[];
  command: string;
  setCommand: (cmd: string) => void;
  handleCommand: () => void;
  isLoading: boolean;
}) => {
  const { mode } = useColorScheme();
  const isDark = mode === "dark" || mode === "system" || !mode;
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop =
        chatContainerRef.current.scrollHeight;
    }
  }, [chatHistory]);

  const handleSubmit = () => {
    handleCommand();
    // Focus the input after sending the message
    setTimeout(() => {
      inputRef.current?.focus();
    }, 0);
  };

  return (
    <Box sx={{ height: "100%", display: "flex", flexDirection: "column", minWidth: 0 }}>
      <Box
        ref={chatContainerRef}
        sx={{
          flex: 1,
          overflowY: "auto",
          overflowX: "hidden",
          p: 2,
          mb: 2,
          backgroundColor:
            isDark
              ? "rgba(13, 5, 8, 0.6)"
              : "rgba(212, 197, 169, 0.4)",
          backgroundImage:
            isDark
              ? "linear-gradient(to bottom, rgba(106, 78, 157, 0.03), rgba(201, 169, 98, 0.03))"
              : "linear-gradient(to bottom, rgba(160, 130, 109, 0.1), rgba(139, 111, 71, 0.1))",
          "&::-webkit-scrollbar": {
            width: "8px",
          },
          "&::-webkit-scrollbar-track": {
            background: "background.default",
            borderRadius: "4px",
          },
          "&::-webkit-scrollbar-thumb": {
            background: "primary.dark",
            borderRadius: "4px",
            "&:hover": {
              background: "primary.main",
            }
          },
        }}
      >
        {chatHistory.map((msg, index) => (
          <ChatMessage key={index} message={msg} />
        ))}
        {isLoading && <LoadingMessage />}
      </Box>

      <Box
        sx={{
          display: "flex",
          gap: 1,
          p: 2,
          borderTop: 2,
          borderColor: "primary.dark",
          backgroundColor:
            isDark
              ? "rgba(26, 15, 30, 0.8)"
              : "rgba(160, 130, 109, 0.3)",
        }}
      >
        <TextField
          inputRef={inputRef}
          fullWidth
          multiline
          maxRows={4}
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyPress={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSubmit();
            }
          }}
          placeholder="Speak thy command..."
          disabled={isLoading}
          size="small"
          autoComplete="off"
          sx={{
            "& .MuiOutlinedInput-root": {
              backgroundColor:
                isDark
                  ? "rgba(62, 44, 46, 0.6)"
                  : "rgba(232, 220, 196, 0.6)",
              fontFamily: "Crimson Text, Georgia, serif",
              fontSize: "1rem",
              "& fieldset": {
                borderColor: "primary.dark",
              },
              "&:hover fieldset": {
                borderColor: "primary.main",
              },
              "&.Mui-focused fieldset": {
                borderColor: "primary.light",
                borderWidth: "2px",
              },
            },
            "& .MuiInputBase-input": {
              color: "text.primary",
            },
            "& .MuiInputBase-input::placeholder": {
              color: "text.secondary",
              opacity: 0.7,
            },
          }}
        />
        <Button
          variant="contained"
          onClick={handleSubmit}
          disabled={isLoading || !command.trim()}
          sx={{
            minWidth: "100px",
            height: "40px", // Match TextField small size height
            fontFamily: "Cinzel, Georgia, serif",
            fontSize: "1rem",
            fontWeight: 600,
          }}
        >
          {isLoading ? <CircularProgress size={20} sx={{ color: "primary.contrastText" }} /> : "Send"}
        </Button>
      </Box>
    </Box>
  );
};
