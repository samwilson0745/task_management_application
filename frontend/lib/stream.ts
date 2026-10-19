import { API_URL, apiRequest } from "./api";

const EVENTS = ["task.created", "task.updated", "task.deleted"];
const RETRY_MS = 3000;

/**
 * Subscribes to live task events over SSE. EventSource cannot send headers, so a
 * short-lived single-purpose ticket is fetched for each (re)connection instead of
 * putting the session token in the URL. Returns a function that closes the stream.
 */
export function connectTaskStream(token: string, onChange: () => void): () => void {
  let source: EventSource | null = null;
  let retry: ReturnType<typeof setTimeout> | null = null;
  let closed = false;

  const connect = async () => {
    try {
      const { ticket } = await apiRequest<{ ticket: string }>("/auth/ticket", {
        method: "POST",
        token,
      });
      if (closed) return;
      source = new EventSource(`${API_URL}/tasks/stream?ticket=${encodeURIComponent(ticket)}`);
      for (const ev of EVENTS) source.addEventListener(ev, onChange);
      source.onerror = () => {
        // The ticket is only valid for the initial handshake, so a dropped
        // connection must be re-established with a fresh one.
        source?.close();
        source = null;
        schedule();
      };
    } catch {
      schedule();
    }
  };

  const schedule = () => {
    if (closed || retry) return;
    retry = setTimeout(() => {
      retry = null;
      connect();
    }, RETRY_MS);
  };

  connect();

  return () => {
    closed = true;
    if (retry) clearTimeout(retry);
    source?.close();
  };
}
