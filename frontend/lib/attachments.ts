import { API_URL, ApiError } from "./api";
import type { TaskAttachment } from "./types";

// Mirrors the backend allowlist: only these types are previewed in a tab.
// Everything else is downloaded, never rendered by the browser.
const PREVIEWABLE = new Set([
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
  "application/pdf",
  "text/plain",
]);

/**
 * Fetches an attachment with the Authorization header (so the session token never
 * appears in a URL) and either previews it in a new tab or saves it.
 */
export async function openAttachment(
  taskId: string,
  attachment: TaskAttachment,
  token: string | null
): Promise<void> {
  // Open the tab synchronously so popup blockers allow it; navigate once loaded.
  const tab = window.open("", "_blank");
  if (tab) tab.opener = null;

  try {
    const res = await fetch(`${API_URL}/tasks/${taskId}/attachments/${attachment.id}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!res.ok) {
      let message = "Failed to open attachment.";
      try {
        message = (await res.json()).error || message;
      } catch {}
      throw new ApiError(res.status, message);
    }

    const type = (res.headers.get("Content-Type") || "").split(";")[0].trim().toLowerCase();
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);

    if (tab && PREVIEWABLE.has(type)) {
      tab.location.href = url;
    } else {
      tab?.close();
      const a = document.createElement("a");
      a.href = url;
      a.download = attachment.filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
    }
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (err) {
    tab?.close();
    throw err;
  }
}
