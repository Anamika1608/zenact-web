import { PROMPT_MAX_LENGTH, PROMPT_MIN_LENGTH } from "./constants";

export function sanitizePrompt(raw: string): {
  valid: boolean;
  value: string;
  error?: string;
} {
  const trimmed = raw.trim();

  if (trimmed.length < PROMPT_MIN_LENGTH) {
    return {
      valid: false,
      value: trimmed,
      error: `Prompt must be at least ${PROMPT_MIN_LENGTH} characters`,
    };
  }

  if (trimmed.length > PROMPT_MAX_LENGTH) {
    return {
      valid: false,
      value: trimmed,
      error: `Prompt must be ${PROMPT_MAX_LENGTH} characters or fewer`,
    };
  }

  // Strip null bytes
  const sanitized = trimmed.replace(/\0/g, "");

  return { valid: true, value: sanitized };
}
