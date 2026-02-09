"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { createTask, getTask, ApiRequestError } from "@/lib/api";
import { TASK_POLL_INTERVAL_MS } from "@/lib/constants";
import { sanitizePrompt } from "@/lib/sanitize";
import type { Task, TaskStatus } from "@/types";

interface UseTaskReturn {
  task: Task | null;
  isCreating: boolean;
  error: string | null;
  submitPrompt: (rawPrompt: string) => Promise<void>;
  reset: () => void;
}

export function useTask(): UseTaskReturn {
  const [task, setTask] = useState<Task | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) clearInterval(pollIntervalRef.current);
    };
  }, []);

  const stopPolling = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  }, []);

  const startPolling = useCallback(
    (taskId: string) => {
      stopPolling();
      pollIntervalRef.current = setInterval(async () => {
        try {
          const updated = await getTask(taskId);
          setTask(updated);
          const terminal: TaskStatus[] = ["completed", "failed"];
          if (terminal.includes(updated.status)) {
            stopPolling();
          }
        } catch (err) {
          console.error("Poll error:", err);
        }
      }, TASK_POLL_INTERVAL_MS);
    },
    [stopPolling]
  );

  const submitPrompt = useCallback(
    async (rawPrompt: string) => {
      const { valid, value, error: validationError } = sanitizePrompt(rawPrompt);
      if (!valid) {
        setError(validationError ?? "Invalid prompt");
        return;
      }

      setError(null);
      setIsCreating(true);
      stopPolling();

      try {
        const response = await createTask(value);
        const newTask: Task = {
          id: response.task_id,
          prompt: value,
          status: "pending",
          steps: [],
          created_at: new Date().toISOString(),
        };
        setTask(newTask);
        startPolling(response.task_id);
      } catch (err) {
        if (err instanceof ApiRequestError) {
          setError(err.message);
        } else {
          setError("Failed to create task. Is the server running?");
        }
        setTask(null);
      } finally {
        setIsCreating(false);
      }
    },
    [startPolling, stopPolling]
  );

  const reset = useCallback(() => {
    stopPolling();
    setTask(null);
    setError(null);
    setIsCreating(false);
  }, [stopPolling]);

  return { task, isCreating, error, submitPrompt, reset };
}
