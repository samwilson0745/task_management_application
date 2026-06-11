"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useRequireAuth } from "@/lib/use-require-auth";
import { useAuth } from "@/lib/auth-context";
import { apiRequest, ApiError } from "@/lib/api";
import type { Task, TaskListResponse, TaskStatus } from "@/lib/types";
import TaskCard from "@/components/TaskCard";

const PAGE_SIZE = 10;

export default function TasksPage() {
  return (
    <Suspense fallback={<div className="text-center text-zinc-500 py-12">Loading...</div>}>
      <TasksPageContent />
    </Suspense>
  );
}

function TasksPageContent() {
  const { user, isLoading: authLoading } = useRequireAuth();
  const { token } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  const status = searchParams.get("status") || "";
  const search = searchParams.get("search") || "";
  const sortBy = searchParams.get("sort_by") || "created_at";
  const sortDir = searchParams.get("sort_dir") || "desc";
  const page = parseInt(searchParams.get("page") || "1", 10);

  const [searchInput, setSearchInput] = useState(search);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const updateParams = useCallback(
    (updates: Record<string, string | null>) => {
      const params = new URLSearchParams(searchParams.toString());
      for (const [key, value] of Object.entries(updates)) {
        if (value === null || value === "") {
          params.delete(key);
        } else {
          params.set(key, value);
        }
      }
      router.push(`/tasks?${params.toString()}`);
    },
    [router, searchParams]
  );

  const fetchTasks = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (status) params.set("status", status);
      if (search) params.set("search", search);
      if (sortBy) params.set("sort_by", sortBy);
      if (sortDir) params.set("sort_dir", sortDir);
      params.set("page", String(page));
      params.set("page_size", String(PAGE_SIZE));

      const resp = await apiRequest<TaskListResponse>(`/tasks/?${params.toString()}`, { token });
      setTasks(resp.data);
      setTotal(resp.total);
      setTotalPages(resp.total_pages);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to load tasks.");
    } finally {
      setLoading(false);
    }
  }, [token, status, search, sortBy, sortDir, page]);

  useEffect(() => {
    if (!authLoading && user) {
      fetchTasks();
    }
  }, [authLoading, user, fetchTasks]);

  // Debounce search input -> URL param
  useEffect(() => {
    const handle = setTimeout(() => {
      if (searchInput !== search) {
        updateParams({ search: searchInput || null, page: null });
      }
    }, 350);
    return () => clearTimeout(handle);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchInput]);

  const handleToggleComplete = async (task: Task) => {
    const newStatus: TaskStatus = task.status === "done" ? "todo" : "done";
    const previous = tasks;
    // Optimistic update
    setTasks((prev) => prev.map((t) => (t.id === task.id ? { ...t, status: newStatus } : t)));
    try {
      await apiRequest<Task>(`/tasks/${task.id}`, {
        method: "PATCH",
        token,
        body: { status: newStatus },
      });
    } catch (err) {
      setTasks(previous);
      setError(err instanceof ApiError ? err.message : "Failed to update task.");
    }
  };

  const handleDelete = async (task: Task) => {
    if (!confirm(`Delete "${task.title}"? This cannot be undone.`)) return;

    const previous = tasks;
    setTasks((prev) => prev.filter((t) => t.id !== task.id));
    try {
      await apiRequest<void>(`/tasks/${task.id}`, { method: "DELETE", token });
      fetchTasks();
    } catch (err) {
      setTasks(previous);
      setError(err instanceof ApiError ? err.message : "Failed to delete task.");
    }
  };

  if (authLoading || !user) {
    return <div className="text-center text-zinc-500 py-12">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">Your Tasks</h1>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:flex-wrap rounded-lg border border-zinc-200 bg-white p-4 shadow-sm">
        <input
          type="search"
          placeholder="Search by title..."
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          className="field w-full sm:w-64"
        />

        <select
          value={status}
          onChange={(e) => updateParams({ status: e.target.value || null, page: null })}
          className="field sm:w-auto"
        >
          <option value="">All statuses</option>
          <option value="todo">To do</option>
          <option value="in_progress">In progress</option>
          <option value="done">Done</option>
        </select>

        <select
          value={sortBy}
          onChange={(e) => updateParams({ sort_by: e.target.value, page: null })}
          className="field sm:w-auto"
        >
          <option value="created_at">Sort by created date</option>
          <option value="due_date">Sort by due date</option>
          <option value="priority">Sort by priority</option>
        </select>

        <select
          value={sortDir}
          onChange={(e) => updateParams({ sort_dir: e.target.value, page: null })}
          className="field sm:w-auto"
        >
          <option value="desc">Descending</option>
          <option value="asc">Ascending</option>
        </select>
      </div>

      {error && (
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <div className="text-center text-zinc-500 py-12">Loading tasks...</div>
      ) : tasks.length === 0 ? (
        <div className="text-center text-zinc-500 py-12 border border-dashed border-zinc-300 rounded-lg">
          <p className="font-medium">No tasks found.</p>
          <p className="text-sm mt-1">
            {search || status
              ? "Try adjusting your filters or search."
              : "Create your first task to get started."}
          </p>
        </div>
      ) : (
        <ul className="space-y-3">
          {tasks.map((task) => (
            <TaskCard
              key={task.id}
              task={task}
              onToggleComplete={handleToggleComplete}
              onDelete={handleDelete}
            />
          ))}
        </ul>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-zinc-500">
            Page {page} of {totalPages} ({total} total)
          </span>
          <div className="flex gap-2">
            <button
              disabled={page <= 1}
              onClick={() => updateParams({ page: String(page - 1) })}
              className="rounded-md border border-zinc-300 px-3 py-1.5 disabled:opacity-50"
            >
              Previous
            </button>
            <button
              disabled={page >= totalPages}
              onClick={() => updateParams({ page: String(page + 1) })}
              className="rounded-md border border-zinc-300 px-3 py-1.5 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
