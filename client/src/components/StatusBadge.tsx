import type { TaskStatus } from "@/types";

const STATUS_CONFIG: Record<
  TaskStatus,
  { label: string; dotColor: string; bgColor: string; textColor: string }
> = {
  pending: {
    label: "Pending",
    dotColor: "bg-yellow-400",
    bgColor: "bg-yellow-400/10",
    textColor: "text-yellow-400",
  },
  running: {
    label: "Running",
    dotColor: "bg-blue-400",
    bgColor: "bg-blue-400/10",
    textColor: "text-blue-400",
  },
  completed: {
    label: "Completed",
    dotColor: "bg-emerald-400",
    bgColor: "bg-emerald-400/10",
    textColor: "text-emerald-400",
  },
  failed: {
    label: "Failed",
    dotColor: "bg-red-400",
    bgColor: "bg-red-400/10",
    textColor: "text-red-400",
  },
};

export function StatusBadge({ status }: { status: TaskStatus }) {
  const config = STATUS_CONFIG[status];

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium ${config.bgColor} ${config.textColor}`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${config.dotColor} ${
          status === "running" ? "animate-pulse" : ""
        }`}
      />
      {config.label}
    </span>
  );
}
