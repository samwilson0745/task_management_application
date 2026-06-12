"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiRequest, API_URL } from "@/lib/api";
import type { Task, TaskAttachment } from "@/lib/types";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

const PRIORITY_STYLES: Record<Task["priority"], string> = {
  low: "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
  medium: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300",
  high: "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300",
};

const STATUS_LABELS: Record<Task["status"], string> = {
  todo: "To do",
  in_progress: "In progress",
  done: "Done",
};

interface TaskCardProps {
  task: Task;
  onToggleComplete: (task: Task) => void;
  onDelete: (task: Task) => void;
  togglingDisabled?: boolean;
  /** When true, the card is shown for another user's task (admin "all tasks" view) and actions are hidden. */
  readOnly?: boolean;
  token?: string | null;
}

export default function TaskCard({ task, onToggleComplete, onDelete, togglingDisabled, readOnly, token }: TaskCardProps) {
  const isDone = task.status === "done";
  const [attachments, setAttachments] = useState<TaskAttachment[]>([]);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    apiRequest<{ data: TaskAttachment[] }>(`/tasks/${task.id}/attachments`, { token })
      .then((resp) => {
        if (!cancelled) setAttachments(resp.data);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [task.id, token]);

  return (
    <li className="rounded-lg border border-zinc-200 bg-white p-4 shadow-sm flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between dark:border-zinc-700 dark:bg-zinc-900">
      <div className="flex items-start gap-3 min-w-0">
        <input
          type="checkbox"
          checked={isDone}
          disabled={togglingDisabled || readOnly}
          onChange={() => onToggleComplete(task)}
          aria-label={isDone ? "Mark as not done" : "Mark as done"}
          className="mt-1 h-4 w-4 rounded border-zinc-300"
        />
        <div className="min-w-0">
          <h3 className={`font-medium break-words ${isDone ? "line-through text-zinc-400" : "text-zinc-900 dark:text-zinc-100"}`}>
            {task.title}
          </h3>
          {task.description && (
            <p className="text-sm text-zinc-500 mt-1 break-words dark:text-zinc-400">{task.description}</p>
          )}
          <div className="flex flex-wrap items-center gap-2 mt-2 text-xs">
            <span className={`px-2 py-0.5 rounded-full font-medium ${PRIORITY_STYLES[task.priority]}`}>
              {task.priority}
            </span>
            <span className="px-2 py-0.5 rounded-full bg-blue-100 text-blue-800 font-medium dark:bg-blue-900/40 dark:text-blue-300">
              {STATUS_LABELS[task.status]}
            </span>
            {task.due_date && (
              <span className="text-zinc-500 dark:text-zinc-400">
                Due {new Date(task.due_date).toLocaleDateString()}
              </span>
            )}
            {task.user_email && (
              <span className="text-zinc-500 dark:text-zinc-400">Owner: {task.user_email}</span>
            )}
          </div>
          {attachments.length > 0 && (
            <ul className="mt-2 space-y-1">
              {attachments.map((a) => (
                <li key={a.id} className="text-xs">
                  <a
                    href={`${API_URL}/tasks/${task.id}/attachments/${a.id}?token=${encodeURIComponent(token || "")}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:underline dark:text-blue-400"
                    title={a.filename}
                  >
                    📎 {a.filename}
                  </a>
                  <span className="text-zinc-400"> ({formatBytes(a.size_bytes)})</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {!readOnly && (
        <div className="flex gap-2 shrink-0">
          <Link
            href={`/tasks/${task.id}`}
            className="rounded-md border border-zinc-300 px-3 py-1.5 text-sm hover:bg-zinc-50 dark:border-zinc-600 dark:hover:bg-zinc-800"
          >
            Edit
          </Link>
          <button
            onClick={() => onDelete(task)}
            className="rounded-md border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 dark:border-red-900 dark:hover:bg-red-950"
          >
            Delete
          </button>
        </div>
      )}
    </li>
  );
}
