/**
 * useGameSocket.ts
 * Manages the WebSocket connection lifecycle for a game session.
 * Dispatches incoming frames to the Zustand store.
 */
import { useEffect, useRef, useCallback } from "react";
import { useGameStore } from "../store/gameStore";
import { getStoredTokens } from "../services/auth.service";
import type { WsFrame, StateDelta, GameStateView, NarrativeChunkPayload, WorldGenLogPayload } from "../types/types";


const WS_ENDPOINT = import.meta.env.VITE_WS_ENDPOINT as string | undefined;

function getWsEndpoint(sessionId: string): string {
  const tokens = getStoredTokens();
  const token = tokens?.accessToken ?? "";
  // Use env var if set; otherwise derive from current host (for local dev)
  const base = WS_ENDPOINT ?? `wss://${window.location.host}/ws`;
  return `${base}?token=${encodeURIComponent(token)}&gameId=${encodeURIComponent(sessionId)}`;
}

interface UseGameSocketOptions {
  sessionId: string;
  onWorldReady?: () => void;
}

export function useGameSocket({ sessionId, onWorldReady }: UseGameSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const retryCount = useRef(0);
  const retryTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMounted = useRef(true);
  const onWorldReadyRef = useRef(onWorldReady);
  onWorldReadyRef.current = onWorldReady;

  const {
    setWsStatus,
    setWsError,
    setStreaming,
    appendStreamChunk,
    finalizeStreamingMessage,
    applyDelta,
    setGameState,
    appendWorldGenLog,
    setWorldGenReady,
  } = useGameStore();

  const handleMessage = useCallback((event: MessageEvent) => {
    let frame: WsFrame;
    try {
      frame = JSON.parse(event.data as string) as WsFrame;
    } catch {
      return;
    }

    switch (frame.type) {
      case "narrative_chunk":
        setStreaming(true);
        appendStreamChunk((frame.payload as NarrativeChunkPayload).content ?? "");
        break;

      case "narrative_end":
        finalizeStreamingMessage();
        break;

      case "game_state_update":
        setGameState(frame.payload as GameStateView);
        break;

      case "state_delta":
        applyDelta(frame.payload as StateDelta);
        break;

      case "streaming_blocked":
        // Server rejected message — user already informed by disabled input
        break;

      case "world_gen_log":
        appendWorldGenLog((frame.payload as WorldGenLogPayload).line ?? "");
        break;

      case "world_gen_ready":
        setWorldGenReady();
        onWorldReadyRef.current?.();
        break;

      case "error":
        setWsError(
          (frame.payload as { message: string })?.message ?? "Unknown error"
        );
        break;
    }
  }, [appendStreamChunk, applyDelta, appendWorldGenLog, finalizeStreamingMessage, setGameState, setStreaming, setWsError, setWorldGenReady]);

  const connect = useCallback(() => {
    if (!isMounted.current) return;
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setWsStatus("connecting");
    const ws = new WebSocket(getWsEndpoint(sessionId));
    wsRef.current = ws;

    ws.onopen = () => {
      if (!isMounted.current) return;
      retryCount.current = 0;
      setWsStatus("connected");
      setWsError(null);
    };

    ws.onmessage = handleMessage;

    ws.onerror = () => {
      setWsStatus("error");
    };

    ws.onclose = () => {
      if (!isMounted.current) return;
      setWsStatus("disconnected");
      // Exponential backoff: 1s, 2s, 4s, cap at 16s, max 5 retries
      if (retryCount.current < 5) {
        const delay = Math.min(1000 * Math.pow(2, retryCount.current), 16000);
        retryCount.current++;
        retryTimer.current = setTimeout(connect, delay);
      }
    };
  }, [sessionId, handleMessage, setWsStatus, setWsError]);

  // Send a chat message through the WebSocket
  const sendChat = useCallback((content: string) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify({ action: "chat", content }));
  }, []);

  // Send a game action (move, pick_up, drop)
  const sendAction = useCallback((subAction: string, payload: string) => {
    if (wsRef.current?.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(
      JSON.stringify({ action: "game_action", sub_action: subAction, payload })
    );
  }, []);

  useEffect(() => {
    isMounted.current = true;
    connect();
    return () => {
      isMounted.current = false;
      if (retryTimer.current) clearTimeout(retryTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [connect]);

  return { sendChat, sendAction };
}
