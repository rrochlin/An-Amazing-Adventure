import { Box, Button, CircularProgress, Paper, TextField } from "@mui/material";
import { useEffect, useRef } from "react";
import { type ChatMessageType } from "../types/types";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

const ChatMessage = ({ message }: { message: ChatMessageType }) => {
  const isPlayer = message.type === "player";
  return (
    <Box
      sx={{
        display: "flex",
        justifyContent: isPlayer ? "flex-end" : "flex-start",
        mb: 2,
      }}
    >
      <Paper
        sx={{
          p: 2,
          maxWidth: "min(600px, 85%)",
          backgroundColor: isPlayer ? "#2196F3" : "#424242",
          color: isPlayer ? "white" : "#E0E0E0",
          borderRadius: 2,
          boxShadow: 1,
          wordWrap: "break-word",
          overflowWrap: "break-word",
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
    <Box sx={{ height: "100%", display: "flex", flexDirection: "column" }}>
      <Box
        ref={chatContainerRef}
        sx={{
          flex: 1,
          overflowY: "auto",
          overflowX: "hidden",
          p: 2,
          mb: 2,
          backgroundColor: "#1E1E1E",
          "&::-webkit-scrollbar": {
            width: "8px",
          },
          "&::-webkit-scrollbar-track": {
            background: "#2D2D2D",
            borderRadius: "4px",
          },
          "&::-webkit-scrollbar-thumb": {
            background: "#424242",
            borderRadius: "4px",
          },
        }}
      >
        {chatHistory.map((msg, index) => (
          <ChatMessage key={index} message={msg} />
        ))}
      </Box>

      <Box
        sx={{
          display: "flex",
          gap: 1,
          p: 2,
          borderTop: 1,
          borderColor: "divider",
          backgroundColor: "#2D2D2D",
        }}
      >
        <TextField
          inputRef={inputRef}
          fullWidth
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyPress={(e) => e.key === "Enter" && handleSubmit()}
          placeholder="Type your command..."
          disabled={isLoading}
          size="small"
          autoComplete="off"
          sx={{
            "& .MuiOutlinedInput-root": {
              backgroundColor: "#424242",
              "& fieldset": {
                borderColor: "#666",
              },
              "&:hover fieldset": {
                borderColor: "#888",
              },
              "&.Mui-focused fieldset": {
                borderColor: "#2196F3",
              },
            },
            "& .MuiInputBase-input": {
              color: "#E0E0E0",
            },
            "& .MuiInputLabel-root": {
              color: "#888",
            },
          }}
        />
        <Button
          variant="contained"
          onClick={handleSubmit}
          disabled={isLoading || !command.trim()}
          sx={{ minWidth: "100px" }}
        >
          {isLoading ? <CircularProgress size={24} /> : "Send"}
        </Button>
      </Box>
    </Box>
  );
};
