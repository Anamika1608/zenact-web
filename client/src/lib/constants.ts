function getApiBaseUrl(): string {
  const url = process.env.NEXT_PUBLIC_API_URL;
  if (!url) {
    // Default fallback — set NEXT_PUBLIC_API_URL in .env.local for production
    return "http://localhost:8080";
  }
  return url.replace(/\/+$/, "");
}

export const API_BASE_URL = getApiBaseUrl();

export function getWsBaseUrl(): string {
  return API_BASE_URL.replace(/^http/, "ws");
}

export const PROMPT_MAX_LENGTH = 2000;
export const PROMPT_MIN_LENGTH = 3;
export const TASK_POLL_INTERVAL_MS = 2000;
export const WS_RECONNECT_DELAY_MS = 1000;
export const WS_MAX_RECONNECT_ATTEMPTS = 5;
