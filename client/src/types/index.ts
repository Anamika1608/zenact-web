// Mirrors Go types from server/models/types.go

export type TaskStatus = "pending" | "running" | "completed" | "failed";

export type ActionType =
  | "navigate"
  | "click"
  | "type"
  | "scroll"
  | "wait"
  | "done"
  | "hold"
  | "drag";

export interface Action {
  action: ActionType; // Go json tag is "action", not "type"
  selector: string;
  value: string;
  done: boolean;
  success: boolean;
}

export interface Step {
  iteration: number;
  screenshot: string; // base64 — empty from REST, present from WS
  url: string;
  title: string;
  thought: string;
  action: Action;
  timestamp: string;
}

export interface Task {
  id: string;
  prompt: string;
  status: TaskStatus;
  steps: Step[];
  error?: string;
  created_at: string;
  completed_at?: string;
}

export interface CreateTaskRequest {
  prompt: string;
}

export interface CreateTaskResponse {
  task_id: string;
  status: string;
}

export type WSEventType =
  | "screenshot"
  | "step_complete"
  | "task_complete"
  | "task_failed";

export interface WSEvent {
  type: WSEventType;
  task_id: string;
  step?: Step;
  screenshot?: string;
  error?: string;
  message?: string;
}

export interface ApiError {
  error: string;
}
