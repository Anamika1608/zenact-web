"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { getWsBaseUrl, WS_RECONNECT_DELAY_MS, WS_MAX_RECONNECT_ATTEMPTS } from "@/lib/constants";
import type { WSEvent, Step } from "@/types";

interface UseTaskWebSocketReturn {
  screenshot: string | null;
  steps: Step[];
  isConnected: boolean;
  completionMessage: string | null;
  failureError: string | null;
}

export function useTaskWebSocket(taskId: string | null): UseTaskWebSocketReturn {
  const [screenshot, setScreenshot] = useState<string | null>(null);
  const [steps, setSteps] = useState<Step[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [completionMessage, setCompletionMessage] = useState<string | null>(null);
  const [failureError, setFailureError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isTerminalRef = useRef(false);

  // Reset state when taskId changes
  useEffect(() => {
    setScreenshot(null);
    setSteps([]);
    setCompletionMessage(null);
    setFailureError(null);
    isTerminalRef.current = false;
    reconnectAttemptsRef.current = 0;
  }, [taskId]);

  const handleEvent = useCallback((event: WSEvent) => {
    switch (event.type) {
      case "screenshot":
        if (event.screenshot) setScreenshot(event.screenshot);
        break;

      case "step_complete":
        if (event.step) {
          if (event.step.screenshot) setScreenshot(event.step.screenshot);
          setSteps((prev) => {
            if (prev.some((s) => s.iteration === event.step!.iteration)) return prev;
            return [...prev, event.step!];
          });
        }
        break;

      case "task_complete":
        setCompletionMessage(event.message ?? "Task completed");
        isTerminalRef.current = true;
        break;

      case "task_failed":
        setFailureError(event.error ?? "Task failed");
        isTerminalRef.current = true;
        break;
    }
  }, []);

  const connect = useCallback(() => {
    if (!taskId) return;

    const wsUrl = `${getWsBaseUrl()}/api/task/${encodeURIComponent(taskId)}/ws`;

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event: MessageEvent) => {
        try {
          const data: WSEvent = JSON.parse(event.data);
          handleEvent(data);
        } catch (err) {
          console.error("Failed to parse WebSocket message:", err);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        wsRef.current = null;

        if (
          !isTerminalRef.current &&
          reconnectAttemptsRef.current < WS_MAX_RECONNECT_ATTEMPTS
        ) {
          const delay = WS_RECONNECT_DELAY_MS * Math.pow(2, reconnectAttemptsRef.current);
          reconnectAttemptsRef.current += 1;
          reconnectTimeoutRef.current = setTimeout(() => connect(), delay);
        }
      };

      ws.onerror = (err) => {
        console.error("WebSocket error:", err);
      };
    } catch (err) {
      console.error("Failed to create WebSocket:", err);
    }
  }, [taskId, handleEvent]);

  useEffect(() => {
    if (!taskId) return;

    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [taskId, connect]);

  return { screenshot, steps, isConnected, completionMessage, failureError };
}
