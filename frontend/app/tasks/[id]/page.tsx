"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useRequireAuth } from "@/lib/use-require-auth";
import { useAuth } from "@/lib/auth-context";
import { apiRequest, ApiError } from "@/lib/api";
import { useToast } from "@/lib/toast-context";
import type { Task } from "@/lib/types";
import TaskForm, { TaskFormValues } from "@/components/TaskForm";
import TaskActivityAndAttachments from "@/components/TaskActivityAndAttachments";

export default function EditTaskPage() {
  const { user, isLoading: authLoading } = useRequireAuth();
  const { token } = useAuth();
  const { showToast } = useToast();
  const router = useRouter();
  const params = useParams<{ id: string }>();

  const [task, setTask] = useState<Task | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authLoading || !user) return;

    apiRequest<Task>(`/tasks/${params.id}`, { token })
      .then(setTask)
      .catch((err) => {
        setError(err instanceof ApiError ? err.message : "Failed to load task.");
      })
      .finally(() => setLoading(false));
  }, [authLoading, user, token, params.id]);

  if (authLoading || !user) {
    return <div className="text-center text-zinc-500 py-12">Loading...</div>;
  }

  if (loading) {
    return <div className="text-center text-zinc-500 py-12">Loading task...</div>;
  }

  if (error || !task) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
        {error || "Task not found."}
      </div>
    );
  }

  const handleSubmit = async (values: TaskFormValues) => {
    await apiRequest<Task>(`/tasks/${task.id}`, {
      method: "PATCH",
      token,
      body: {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        due_date: values.due_date ? new Date(values.due_date).toISOString() : null,
        clear_due_date: !values.due_date,
      },
    });
    showToast("Task updated.");
    router.push("/tasks");
  };

  const handleDelete = async () => {
    await apiRequest<void>(`/tasks/${task.id}`, { method: "DELETE", token });
    showToast("Task deleted.");
    router.push("/tasks");
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Edit Task</h1>
      <TaskForm initial={task} submitLabel="Save changes" onSubmit={handleSubmit} onDelete={handleDelete} />
      <TaskActivityAndAttachments taskId={task.id} token={token} />
    </div>
  );
}
