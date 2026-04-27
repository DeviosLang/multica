export function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function generateUUID(): string {
  const cryptoObj = globalThis.crypto;

  if (!cryptoObj?.getRandomValues) {
    throw new Error("Secure UUID generation requires crypto.getRandomValues");
  }

  const bytes = new Uint8Array(16);
  cryptoObj.getRandomValues(bytes);

  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40; // version 4
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80; // variant 1

  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");

  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

/**
 * Generate an id that prefers crypto.randomUUID but falls back in non-secure contexts.
 */
export function createSafeId(): string {
  const cryptoObj = globalThis.crypto;

  if (cryptoObj?.randomUUID) {
    try {
      return cryptoObj.randomUUID();
    } catch {
      // Fall through to fallback.
    }
  }

  return generateUUID();
}

/** Request id helper used for logs/tracing headers. */
export function createRequestId(length = 8): string {
  return createSafeId().replace(/-/g, "").slice(0, length);
}

/**
 * Copy text to the clipboard.
 *
 * Uses the modern Clipboard API when available (requires HTTPS or localhost).
 * Falls back to the legacy execCommand approach so HTTP deployments still work.
 *
 * Returns true on success, false on failure.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  // Modern Clipboard API (HTTPS / localhost only)
  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Fall through to legacy approach
    }
  }

  // Legacy execCommand fallback (works over HTTP)
  if (typeof document !== "undefined") {
    try {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      const success = document.execCommand("copy");
      document.body.removeChild(textarea);
      return success;
    } catch {
      return false;
    }
  }

  return false;
}
