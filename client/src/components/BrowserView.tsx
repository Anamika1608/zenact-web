import type { TaskStatus } from "@/types";

interface BrowserViewProps {
  screenshot: string | null;
  status: TaskStatus | null;
  url?: string;
}

export function BrowserView({ screenshot, status, url }: BrowserViewProps) {
  return (
    <div className="flex h-full flex-col overflow-hidden rounded-xl border border-zinc-700/50 bg-zinc-900">
      {/* Browser chrome bar */}
      <div className="flex items-center gap-2 border-b border-zinc-700/50 bg-zinc-800/50 px-4 py-2.5">
        <div className="flex gap-1.5">
          <div className="h-3 w-3 rounded-full bg-zinc-600" />
          <div className="h-3 w-3 rounded-full bg-zinc-600" />
          <div className="h-3 w-3 rounded-full bg-zinc-600" />
        </div>
        <div className="ml-2 flex-1 rounded-md bg-zinc-700/50 px-3 py-1">
          <span className="text-xs text-zinc-400 select-all">
            {url || "about:blank"}
          </span>
        </div>
        {status === "running" && (
          <div className="flex items-center gap-1.5">
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75" />
              <span className="relative inline-flex h-2 w-2 rounded-full bg-blue-500" />
            </span>
            <span className="text-xs text-zinc-500">Live</span>
          </div>
        )}
      </div>

      {/* Screenshot viewport */}
      <div className="relative flex-1 bg-zinc-950">
        {screenshot ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={`data:image/png;base64,${screenshot}`}
            alt="Browser screenshot"
            className="h-full w-full object-contain"
            draggable={false}
          />
        ) : (
          <div className="flex h-full items-center justify-center">
            {status === "pending" || status === "running" ? (
              <div className="flex flex-col items-center gap-3 text-zinc-600">
                <svg className="h-8 w-8 animate-spin" viewBox="0 0 24 24" fill="none">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                <span className="text-sm">
                  {status === "pending" ? "Starting browser..." : "Waiting for screenshot..."}
                </span>
              </div>
            ) : (
              <div className="flex flex-col items-center gap-2 text-zinc-600">
                <svg className="h-12 w-12" fill="none" viewBox="0 0 24 24" strokeWidth="1" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 17.25v1.007a3 3 0 01-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0115 18.257V17.25m6-12V15a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 15V5.25m18 0A2.25 2.25 0 0018.75 3H5.25A2.25 2.25 0 003 5.25m18 0V12a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 12V5.25" />
                </svg>
                <span className="text-sm">Enter a prompt to start the browser agent</span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
