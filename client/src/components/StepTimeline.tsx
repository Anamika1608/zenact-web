"use client";

import { useEffect, useRef } from "react";
import type { Step } from "@/types";

interface StepTimelineProps {
  steps: Step[];
  completionMessage: string | null;
  failureError: string | null;
}

function ActionIcon({ action }: { action: string }) {
  const cls = "h-3.5 w-3.5";
  switch (action) {
    case "navigate":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" />
        </svg>
      );
    case "click":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59" />
        </svg>
      );
    case "type":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5M12 17.25h8.25" />
        </svg>
      );
    case "scroll":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 7.5L7.5 3m0 0L12 7.5M7.5 3v13.5m13.5-4.5L16.5 16.5m0 0L12 12m4.5 4.5V3" />
        </svg>
      );
    case "wait":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    case "done":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    case "hold":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
        </svg>
      );
    case "drag":
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3v11.25A2.25 2.25 0 006 16.5h2.25M3.75 3h-1.5m1.5 0h16.5m0 0h1.5m-1.5 0v11.25A2.25 2.25 0 0118 16.5h-2.25m-7.5 0h7.5m-7.5 0l-1 3m8.5-3l1 3m0 0l.5 1.5m-.5-1.5h-9.5m0 0l-.5 1.5M9 11.25v1.5M12 9v3.75m3-6v6" />
        </svg>
      );
    default:
      return (
        <svg className={cls} fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
        </svg>
      );
  }
}

function formatActionLabel(step: Step): string {
  const { action } = step;
  switch (action.action) {
    case "navigate":
      return `Navigate to ${action.value}`;
    case "click":
      return `Click ${action.selector}`;
    case "type":
      return `Type "${action.value}" into ${action.selector}`;
    case "scroll":
      return `Scroll ${action.value || "down"}`;
    case "wait":
      return "Wait for page";
    case "done":
      return "Task complete";
    case "hold":
      return `Hold ${action.selector}${action.value ? ` for ${action.value}ms` : ""}`;
    case "drag":
      return `Drag ${action.selector} to ${action.value}`;
    default:
      return action.action;
  }
}

export function StepTimeline({ steps, completionMessage, failureError }: StepTimelineProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [steps.length, completionMessage, failureError]);

  if (steps.length === 0 && !completionMessage && !failureError) {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <p className="text-sm text-zinc-600">Agent steps will appear here as the task runs.</p>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="space-y-1">
        {steps.map((step) => (
          <div key={step.iteration} className="group relative pl-6">
            {/* Timeline line */}
            <div className="absolute left-[9px] top-6 h-full w-px bg-zinc-700/50 group-last:hidden" />
            {/* Timeline dot */}
            <div
              className={`absolute left-0 top-1.5 flex h-[18px] w-[18px] items-center justify-center rounded-full border ${
                step.action.done
                  ? "border-emerald-500/50 bg-emerald-500/20 text-emerald-400"
                  : "border-zinc-600 bg-zinc-800 text-zinc-400"
              }`}
            >
              <ActionIcon action={step.action.action} />
            </div>

            <div className="rounded-lg pb-4">
              <div className="flex items-baseline gap-2">
                <span className="text-xs font-medium text-zinc-500">Step {step.iteration}</span>
                <span className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-xs text-zinc-400">
                  {step.action.action}
                </span>
              </div>
              <p className="mt-1 text-sm leading-relaxed text-zinc-300">{step.thought}</p>
              <p className="mt-1 font-mono text-xs text-zinc-500">{formatActionLabel(step)}</p>
              {step.url && (
                <p className="mt-0.5 truncate text-xs text-zinc-600">
                  {step.title ? `${step.title} — ` : ""}{step.url}
                </p>
              )}
            </div>
          </div>
        ))}

        {completionMessage && (
          <div className="relative pl-6">
            <div className="absolute left-0 top-1.5 flex h-[18px] w-[18px] items-center justify-center rounded-full border border-emerald-500/50 bg-emerald-500/20 text-emerald-400">
              <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div className="rounded-lg border border-emerald-500/10 bg-emerald-500/5 p-3">
              <p className="text-sm font-medium text-emerald-400">{completionMessage}</p>
            </div>
          </div>
        )}

        {failureError && (
          <div className="relative pl-6">
            <div className="absolute left-0 top-1.5 flex h-[18px] w-[18px] items-center justify-center rounded-full border border-red-500/50 bg-red-500/20 text-red-400">
              <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
            </div>
            <div className="rounded-lg border border-red-500/10 bg-red-500/5 p-3">
              <p className="text-sm font-medium text-red-400">{failureError}</p>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>
    </div>
  );
}
