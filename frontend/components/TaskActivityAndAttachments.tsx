"use client";

import { ChangeEvent, useEffect, useState } from "react";
import { apiRequest, apiUpload, ApiError, API_URL } from "@/lib/api";
import { useToast } from "@/lib/toast-context";
import type { TaskActivity, TaskAttachment } from "@/lib/types";

const MAX_FILE_SIZE = 10 * 1024 * 1024;

const ACTION_LABELS: Record<string, string> = {
  created: "Task created",
  updated: "Task updated",
};

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface Props {
  taskId: string;
  token: string | null;
}

export default function TaskActivityAndAttachments({ taskId, token }: Props) {
  const { showToast } = useToast();

  const [attachments, setAttachments] = useState<TaskAttachment[]>([]);
  const [activity, setActivity] = useState<TaskActivity[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    if (!token) return;
    try {
      const [attachmentsResp, activityResp] = await Promise.all([
        apiRequest<{ data: TaskAttachment[] }>(`/tasks/${taskId}/attachments`, { token }),
        apiRequest<{ data: TaskActivity[] }>(`/tasks/${taskId}/activity`, { token }),
      ]);
      setAttachments(attachmentsResp.data);
      setActivity(activityResp.data);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to load task details.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [taskId, token]);

  const handleUpload = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;

    if (file.size > MAX_FILE_SIZE) {
      showToast("File exceeds 10MB limit.", "error");
      return;
    }

    setUploading(true);
    try {
      await apiUpload<TaskAttachment>(`/tasks/${taskId}/attachments`, file, token);
      showToast("File uploaded.");
      await load();
    } catch (err) {
      showToast(err instanceof ApiError ? err.message : "Upload failed.", "error");
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (attachment: TaskAttachment) => {
    if (!confirm(`Delete "${attachment.filename}"?`)) return;
    const previous = attachments;
    setAttachments((prev) => prev.filter((a) => a.id !== attachment.id));
    try {
      await apiRequest<void>(`/tasks/${taskId}/attachments/${attachment.id}`, { method: "DELETE", token });
      showToast("Attachment deleted.");
    } catch (err) {
      setAttachments(previous);
      showToast(err instanceof ApiError ? err.message : "Failed to delete attachment.", "error");
    }
  };

  if (loading) {
    return <div className="text-sm text-zinc-500 dark:text-zinc-400">Loading attachments and activity...</div>;
  }

  if (error) {
    return (
      <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
        {error}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 max-w-3xl">
      <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm dark:border-zinc-700 dark:bg-zinc-900">
        <h2 className="text-sm font-semibold mb-3">Attachments</h2>

        <label className="inline-block">
          <span className="rounded-md border border-zinc-300 px-3 py-1.5 text-sm cursor-pointer hover:bg-zinc-50 dark:border-zinc-600 dark:hover:bg-zinc-800">
            {uploading ? "Uploading..." : "Upload file"}
          </span>
          <input type="file" className="hidden" onChange={handleUpload} disabled={uploading} />
        </label>
        <p className="text-xs text-zinc-500 mt-1 dark:text-zinc-400">Max 10MB per file.</p>

        {attachments.length === 0 ? (
          <p className="text-sm text-zinc-500 mt-3 dark:text-zinc-400">No attachments yet.</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {attachments.map((a) => (
              <li key={a.id} className="flex items-center justify-between gap-2 text-sm">
                <a
                  href={`${API_URL}/tasks/${taskId}/attachments/${a.id}?token=${encodeURIComponent(token || "")}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="truncate text-zinc-700 hover:underline dark:text-zinc-300"
                  title={a.filename}
                >
                  {a.filename}
                </a>
                <div className="flex items-center gap-2 shrink-0">
                  <span className="text-xs text-zinc-400">{formatBytes(a.size_bytes)}</span>
                  <button
                    onClick={() => handleDelete(a)}
                    className="text-xs text-red-600 hover:underline"
                  >
                    Delete
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm dark:border-zinc-700 dark:bg-zinc-900">
        <h2 className="text-sm font-semibold mb-3">Activity</h2>
        {activity.length === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">No activity yet.</p>
        ) : (
          <ul className="space-y-2">
            {activity.map((a) => (
              <li key={a.id} className="text-sm text-zinc-600 dark:text-zinc-300">
                <span className="font-medium">{ACTION_LABELS[a.action] || a.action}</span>
                {a.user_email && <span className="text-zinc-400"> by {a.user_email}</span>}
                <div className="text-xs text-zinc-400">{new Date(a.created_at).toLocaleString()}</div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
