"use client";

import { useCallback } from "react";
import { useTask } from "@/hooks/useTask";
import { useTaskWebSocket } from "@/hooks/useTaskWebSocket";
import { TaskInput } from "@/components/TaskInput";
import { BrowserView } from "@/components/BrowserView";
import { StepTimeline } from "@/components/StepTimeline";
import { StatusBadge } from "@/components/StatusBadge";
import { ErrorBanner } from "@/components/ErrorBanner";

export default function HomePage() {
  const { task, isCreating, error, submitPrompt, reset } = useTask();

  const {
    screenshot,
    steps: wsSteps,
    isConnected,
    completionMessage,
    failureError,
  } = useTaskWebSocket(task?.id ?? null);

  // Prefer WebSocket steps (they have screenshots), fall back to polled task steps
  const steps = wsSteps.length > 0 ? wsSteps : task?.steps ?? [];

  // WS terminal events arrive faster than next poll cycle
  const effectiveStatus = completionMessage
    ? ("completed" as const)
    : failureError
      ? ("failed" as const)
      : task?.status ?? null;

  const latestUrl = steps.length > 0 ? steps[steps.length - 1].url : undefined;

  const handleNewTask = useCallback(() => {
    reset();
  }, [reset]);

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-zinc-950">
      {/* Header */}
      <header className="flex items-center justify-between border-b border-zinc-800 px-6 py-3">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold tracking-tight text-zinc-100">Zenact</h1>
          <span className="rounded-md bg-zinc-800 px-2 py-0.5 text-xs text-zinc-500">Agent</span>
        </div>
        <div className="flex items-center gap-3">
          {effectiveStatus && <StatusBadge status={effectiveStatus} />}
          {task && (
            <button
              onClick={handleNewTask}
              className="rounded-lg border border-zinc-700 px-3 py-1.5 text-xs text-zinc-400 transition-colors hover:border-zinc-600 hover:text-zinc-300"
            >
              New Task
            </button>
          )}
        </div>
      </header>

      {/* Error banner */}
      {(error || failureError) && (
        <div className="px-6 pt-4">
          <ErrorBanner
            message={error || failureError || "An error occurred"}
            onDismiss={error ? () => reset() : undefined}
          />
        </div>
      )}

      {/* Main content */}
      {!task ? (
        /* Initial state: centered prompt input */
        <div className="flex flex-1 items-center justify-center p-6">
          <div className="w-full max-w-2xl">
            <div className="mb-8 text-center">
              <h2 className="text-2xl font-semibold tracking-tight text-zinc-100">
                What should the agent do?
              </h2>
              <p className="mt-2 text-sm text-zinc-500">
                Describe a browser task in natural language. The agent will navigate, click, type, and scroll to complete it.
              </p>
            </div>
            <TaskInput onSubmit={submitPrompt} isLoading={isCreating} />
          </div>
        </div>
      ) : (
        /* Active task: split view */
        <div className="flex flex-1 overflow-hidden">
          {/* Left: Browser view */}
          <div className="flex flex-1 flex-col overflow-hidden p-4">
            <div className="mb-3 flex items-start gap-2">
              <span className="mt-0.5 shrink-0 text-xs font-medium uppercase tracking-wider text-zinc-600">
                Task
              </span>
              <p className="line-clamp-2 text-sm text-zinc-300">{task.prompt}</p>
            </div>
            <div className="min-h-0 flex-1">
              <BrowserView screenshot={screenshot} status={effectiveStatus} url={latestUrl} />
            </div>
          </div>

          {/* Right: Step timeline */}
          <div className="flex w-96 shrink-0 flex-col overflow-hidden border-l border-zinc-800">
            <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-3">
              <h3 className="text-sm font-medium text-zinc-300">Agent Steps</h3>
              <span className="text-xs text-zinc-600">
                {steps.length} step{steps.length !== 1 ? "s" : ""}
              </span>
            </div>
            <div className="min-h-0 flex-1 overflow-hidden">
              <StepTimeline
                steps={steps}
                completionMessage={completionMessage}
                failureError={failureError}
              />
            </div>
          </div>
        </div>
      )}

      {/* WebSocket connection indicator */}
      {task && (
        <div className="absolute bottom-3 left-3">
          <div
            className={`flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs ${
              isConnected ? "bg-emerald-500/10 text-emerald-500" : "bg-zinc-800 text-zinc-500"
            }`}
          >
            <span
              className={`h-1.5 w-1.5 rounded-full ${isConnected ? "bg-emerald-500" : "bg-zinc-600"}`}
            />
            {isConnected ? "Connected" : "Reconnecting..."}
          </div>
        </div>
      )}
    </div>
  );
}
