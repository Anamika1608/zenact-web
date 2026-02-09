"use client";

import { useState, useCallback, type FormEvent, type KeyboardEvent } from "react";
import { PROMPT_MAX_LENGTH } from "@/lib/constants";

interface TaskInputProps {
  onSubmit: (prompt: string) => void;
  isLoading: boolean;
  disabled?: boolean;
}

export function TaskInput({ onSubmit, isLoading, disabled }: TaskInputProps) {
  const [prompt, setPrompt] = useState("");

  const handleSubmit = useCallback(
    (e: FormEvent) => {
      e.preventDefault();
      const trimmed = prompt.trim();
      if (!trimmed || isLoading || disabled) return;
      onSubmit(trimmed);
    },
    [prompt, isLoading, disabled, onSubmit]
  );

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        const trimmed = prompt.trim();
        if (!trimmed || isLoading || disabled) return;
        onSubmit(trimmed);
      }
    },
    [prompt, isLoading, disabled, onSubmit]
  );

  const remaining = PROMPT_MAX_LENGTH - prompt.length;
  const isOverLimit = remaining < 0;

  return (
    <form onSubmit={handleSubmit} className="w-full">
      <div className="relative">
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Describe what you want the browser agent to do..."
          rows={3}
          disabled={isLoading || disabled}
          className="w-full resize-none rounded-xl border border-zinc-700 bg-zinc-800/50 px-4 py-3 text-sm text-zinc-100 placeholder-zinc-500 outline-none transition-colors focus:border-zinc-500 focus:ring-1 focus:ring-zinc-500 disabled:opacity-50"
          aria-label="Task prompt"
        />
        <div className="mt-2 flex items-center justify-between">
          <span
            className={`text-xs ${
              isOverLimit
                ? "text-red-400"
                : remaining < 100
                  ? "text-yellow-400"
                  : "text-zinc-500"
            }`}
          >
            {remaining} characters remaining
          </span>
          <div className="flex items-center gap-3">
            <span className="text-xs text-zinc-600">
              {typeof navigator !== "undefined" && /Mac/i.test(navigator.userAgent)
                ? "\u2318"
                : "Ctrl"}
              +Enter
            </span>
            <button
              type="submit"
              disabled={!prompt.trim() || isLoading || disabled || isOverLimit}
              className="rounded-lg bg-zinc-100 px-4 py-2 text-sm font-medium text-zinc-900 transition-all hover:bg-white disabled:cursor-not-allowed disabled:opacity-40"
            >
              {isLoading ? (
                <span className="flex items-center gap-2">
                  <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Starting...
                </span>
              ) : (
                "Run Agent"
              )}
            </button>
          </div>
        </div>
      </div>
    </form>
  );
}
