/**
 * WorldGenTerminal
 * Displays world-generation progress as a scrollable terminal window.
 * Lines arrive over WebSocket as world_gen_log frames.
 * Shows a blinking cursor while generation is in progress.
 */
import { useEffect, useRef } from "react";
import { Box, Typography, LinearProgress } from "@mui/material";

interface WorldGenTerminalProps {
  lines: string[];
  ready: boolean;
}

export function WorldGenTerminal({ lines, ready }: WorldGenTerminalProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom whenever a new line arrives
  useEffect(() => {
    // scrollIntoView may be absent in test environments (jsdom)
    if (typeof bottomRef.current?.scrollIntoView === "function") {
      bottomRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [lines]);

  return (
    <Box
      sx={{
        width: "100%",
        borderRadius: 1,
        overflow: "hidden",
        border: "1px solid rgba(0, 255, 70, 0.3)",
        boxShadow: "0 0 20px rgba(0, 255, 70, 0.1), inset 0 0 40px rgba(0,0,0,0.6)",
        fontFamily: '"Courier New", "Courier", monospace',
      }}
    >
      {/* Terminal title bar */}
      <Box
        sx={{
          px: 2,
          py: 0.75,
          background: "rgba(0, 255, 70, 0.08)",
          borderBottom: "1px solid rgba(0, 255, 70, 0.2)",
          display: "flex",
          alignItems: "center",
          gap: 1,
        }}
      >
        <Box sx={{ width: 10, height: 10, borderRadius: "50%", bgcolor: ready ? "#00ff46" : "#ff9800" }} />
        <Typography
          variant="caption"
          sx={{ color: "rgba(0, 255, 70, 0.7)", fontFamily: "inherit", letterSpacing: "0.15em" }}
        >
          WORLD ARCHITECT — {ready ? "COMPLETE" : "GENERATING..."}
        </Typography>
      </Box>

      {/* Progress bar */}
      {!ready && (
        <LinearProgress
          sx={{
            height: 2,
            "& .MuiLinearProgress-bar": { background: "rgba(0, 255, 70, 0.6)" },
            background: "rgba(0, 255, 70, 0.1)",
          }}
        />
      )}

      {/* Log output */}
      <Box
        sx={{
          p: 2,
          minHeight: 220,
          maxHeight: 360,
          overflowY: "auto",
          background: "rgba(0, 8, 0, 0.85)",
          "&::-webkit-scrollbar": { width: "6px" },
          "&::-webkit-scrollbar-track": { background: "transparent" },
          "&::-webkit-scrollbar-thumb": { background: "rgba(0, 255, 70, 0.3)", borderRadius: "3px" },
        }}
      >
        {lines.length === 0 && (
          <Typography
            component="div"
            sx={{ color: "rgba(0, 255, 70, 0.4)", fontFamily: "inherit", fontSize: "0.85rem" }}
          >
            Waiting for Architect...
          </Typography>
        )}
        {lines.map((line, i) => (
          <Typography
            key={i}
            component="div"
            sx={{
              color: line.startsWith("ERROR")
                ? "#ff4444"
                : line.startsWith("Your adventure")
                ? "#00ff46"
                : "rgba(0, 255, 70, 0.85)",
              fontFamily: "inherit",
              fontSize: "0.82rem",
              lineHeight: 1.6,
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
              "&::before": { content: '"> "', opacity: 0.5 },
            }}
          >
            {line}
          </Typography>
        ))}
        {/* Blinking cursor while in progress */}
        {!ready && (
          <Box
            component="span"
            sx={{
              display: "inline-block",
              width: "8px",
              height: "14px",
              bgcolor: "rgba(0, 255, 70, 0.8)",
              ml: "2px",
              verticalAlign: "middle",
              animation: "blink 1s step-end infinite",
              "@keyframes blink": {
                "0%, 100%": { opacity: 1 },
                "50%": { opacity: 0 },
              },
            }}
          />
        )}
        <div ref={bottomRef} />
      </Box>
    </Box>
  );
}
