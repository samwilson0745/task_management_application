"use client";

import { FormEvent, useState } from "react";
import type { Task, TaskPriority, TaskStatus } from "@/lib/types";

export interface TaskFormValues {
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  due_date: string; // yyyy-mm-dd or ""
  files?: File[];
}

interface TaskFormProps {
  initial?: Task;
  submitLabel: string;
  onSubmit: (values: TaskFormValues) => Promise<void>;
  onDelete?: () => Promise<void>;
}

const MAX_TITLE_LENGTH = 200;
const MAX_DESCRIPTION_LENGTH = 2000;

function toDateInputValue(dueDate: string | null): string {
  if (!dueDate) return "";
  return dueDate.slice(0, 10);
}

function todayInputValue(): string {
  return new Date().toISOString().slice(0, 10);
}

function fieldClass(hasError: boolean) {
  return `field${hasError ? " field-error" : ""}`;
}

export default function TaskForm({ initial, submitLabel, onSubmit, onDelete }: TaskFormProps) {
  const [title, setTitle] = useState(initial?.title ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [status, setStatus] = useState<TaskStatus>(initial?.status ?? "todo");
  const [priority, setPriority] = useState<TaskPriority>(initial?.priority ?? "medium");
  const [dueDate, setDueDate] = useState(toDateInputValue(initial?.due_date ?? null));
  const [files, setFiles] = useState<File[]>([]);

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const validate = (): Record<string, string> => {
    const errs: Record<string, string> = {};
    const trimmedTitle = title.trim();

    if (!trimmedTitle) {
      errs.title = "Title is required.";
    } else if (trimmedTitle.length > MAX_TITLE_LENGTH) {
      errs.title = `Title must be ${MAX_TITLE_LENGTH} characters or fewer.`;
    }

    if (description.length > MAX_DESCRIPTION_LENGTH) {
      errs.description = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or fewer.`;
    }

    if (dueDate) {
      const parsed = new Date(dueDate);
      if (Number.isNaN(parsed.getTime())) {
        errs.due_date = "Enter a valid date.";
      } else if (dueDate < todayInputValue()) {
        errs.due_date = "Due date cannot be in the past.";
      }
    }

    if (!["todo", "in_progress", "done"].includes(status)) {
      errs.status = "Select a valid status.";
    }

    if (!["low", "medium", "high"].includes(priority)) {
      errs.priority = "Select a valid priority.";
    }

    return errs;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitError(null);

    const errs = validate();
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setSubmitting(true);
    try {
      await onSubmit({ title: title.trim(), description: description.trim(), status, priority, due_date: dueDate, files });
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!onDelete) return;
    if (!confirm("Delete this task? This cannot be undone.")) return;
    setDeleting(true);
    try {
      await onDelete();
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to delete task.");
      setDeleting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-5 max-w-xl rounded-lg border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-700 dark:bg-zinc-900" noValidate>
      <div>
        <label htmlFor="title" className="block text-sm font-medium text-zinc-700 mb-1 dark:text-zinc-300">
          Title <span className="text-red-500">*</span>
        </label>
        <input
          id="title"
          type="text"
          placeholder="e.g. Write project proposal"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          aria-invalid={!!errors.title}
          className={fieldClass(!!errors.title)}
        />
        {errors.title && <p className="mt-1 text-sm text-red-600">{errors.title}</p>}
      </div>

      <div>
        <label htmlFor="description" className="block text-sm font-medium text-zinc-700 mb-1 dark:text-zinc-300">
          Description
        </label>
        <textarea
          id="description"
          placeholder="Add more detail about this task (optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={4}
          aria-invalid={!!errors.description}
          className={fieldClass(!!errors.description)}
        />
        {errors.description && <p className="mt-1 text-sm text-red-600">{errors.description}</p>}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div>
          <label htmlFor="status" className="block text-sm font-medium text-zinc-700 mb-1 dark:text-zinc-300">
            Status
          </label>
          <select
            id="status"
            value={status}
            onChange={(e) => setStatus(e.target.value as TaskStatus)}
            className="field"
          >
            <option value="todo">To do</option>
            <option value="in_progress">In progress</option>
            <option value="done">Done</option>
          </select>
        </div>

        <div>
          <label htmlFor="priority" className="block text-sm font-medium text-zinc-700 mb-1 dark:text-zinc-300">
            Priority
          </label>
          <select
            id="priority"
            value={priority}
            onChange={(e) => setPriority(e.target.value as TaskPriority)}
            className="field"
          >
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
          </select>
        </div>

        <div>
          <label htmlFor="due_date" className="block text-sm font-medium text-zinc-700 mb-1 dark:text-zinc-300">
            Due date
          </label>
          <input
            id="due_date"
            type="date"
            min={todayInputValue()}
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
            aria-invalid={!!errors.due_date}
            className={fieldClass(!!errors.due_date)}
          />
          {errors.due_date && <p className="mt-1 text-sm text-red-600">{errors.due_date}</p>}
        </div>
      </div>

      {!initial && (
        <div>
          <label htmlFor="files" className="block text-sm font-medium text-zinc-700 mb-1 dark:text-zinc-300">
            Attachments
          </label>
          <input
            id="files"
            type="file"
            multiple
            onChange={(e) => setFiles(Array.from(e.target.files ?? []))}
            className="block w-full text-sm text-zinc-700 dark:text-zinc-300 file:mr-3 file:rounded-md file:border file:border-zinc-300 file:bg-zinc-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-zinc-700 file:cursor-pointer hover:file:bg-zinc-100 dark:file:border-zinc-600 dark:file:bg-zinc-800 dark:file:text-zinc-200 dark:hover:file:bg-zinc-700"
          />
          <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">Max 10MB per file. Uploaded after the task is created.</p>
          {files.length > 0 && (
            <ul className="mt-2 space-y-1">
              {files.map((file, i) => (
                <li key={i} className="flex items-center justify-between gap-2 text-sm text-zinc-600 dark:text-zinc-300">
                  <span className="truncate" title={file.name}>{file.name}</span>
                  <button
                    type="button"
                    onClick={() => setFiles((prev) => prev.filter((_, idx) => idx !== i))}
                    className="text-xs text-red-600 hover:underline shrink-0"
                  >
                    Remove
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {submitError && (
        <p role="alert" className="text-sm text-red-600">
          {submitError}
        </p>
      )}

      <div className="flex items-center gap-3 pt-1">
        <button
          type="submit"
          disabled={submitting || deleting}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          {submitting ? "Saving..." : submitLabel}
        </button>

        {onDelete && (
          <button
            type="button"
            onClick={handleDelete}
            disabled={submitting || deleting}
            className="rounded-md border border-red-200 px-4 py-2 text-sm text-red-600 hover:bg-red-50 disabled:opacity-50"
          >
            {deleting ? "Deleting..." : "Delete task"}
          </button>
        )}
      </div>
    </form>
  );
}
