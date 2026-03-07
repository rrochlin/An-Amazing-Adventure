/**
 * useWorldGenSocket
 * Lightweight WebSocket hook used only during world generation.
 * Connects immediately after a game is created, collects world_gen_log frames
 * into local state, and fires onReady when world_gen_ready is received.
 * Cleans up automatically on unmount or when onReady fires.
 */
import { useEffect, useRef, useState, useCallback } from "react";
import { getStoredTokens } from "../services/auth.service";
import type { WsFrame, WorldGenLogPayload } from "../types/types";

const WS_ENDPOINT = import.meta.env.VITE_WS_ENDPOINT as string | undefined;

function getWsUrl(sessionId: string): string {
  const tokens = getStoredTokens();
  const token = tokens?.accessToken ?? "";
  const base = WS_ENDPOINT ?? `wss://${window.location.host}/ws`;
  return `${base}?token=${encodeURIComponent(token)}&gameId=${encodeURIComponent(sessionId)}`;
}

interface UseWorldGenSocketOptions {
  sessionId: string | null; // null = not started yet
  onReady: () => void;
}

export function useWorldGenSocket({ sessionId, onReady }: UseWorldGenSocketOptions) {
  const [lines, setLines] = useState<string[]>([]);
  const [ready, setReady] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;

  const appendLine = useCallback((line: string) => {
    setLines((prev) => [...prev, line]);
  }, []);

  useEffect(() => {
    if (!sessionId) return;

    // Reset state for new session
    setLines([]);
    setReady(false);

    const ws = new WebSocket(getWsUrl(sessionId));
    wsRef.current = ws;

    ws.onmessage = (event: MessageEvent) => {
      let frame: WsFrame;
      try {
        frame = JSON.parse(event.data as string) as WsFrame;
      } catch {
        return;
      }
      if (frame.type === "world_gen_log") {
        appendLine((frame.payload as WorldGenLogPayload).line ?? "");
      } else if (frame.type === "world_gen_ready") {
        setReady(true);
        ws.close();
        // Small delay so user sees the final log lines before navigating
        setTimeout(() => onReadyRef.current(), 800);
      }
    };

    ws.onerror = () => {
      appendLine("ERROR: connection lost — refresh to retry");
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [sessionId, appendLine]);

  return { lines, ready };
}
