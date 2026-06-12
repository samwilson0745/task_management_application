"use client";

import { useRouter } from "next/navigation";
import { useRequireAuth } from "@/lib/use-require-auth";
import { useAuth } from "@/lib/auth-context";
import { apiRequest, apiUpload } from "@/lib/api";
import { useToast } from "@/lib/toast-context";
import type { Task } from "@/lib/types";
import TaskForm, { TaskFormValues } from "@/components/TaskForm";

export default function NewTaskPage() {
  const { user, isLoading } = useRequireAuth();
  const { token } = useAuth();
  const { showToast } = useToast();
  const router = useRouter();

  if (isLoading || !user) {
    return <div className="text-center text-zinc-500 py-12">Loading...</div>;
  }

  const handleSubmit = async (values: TaskFormValues) => {
    const task = await apiRequest<Task>("/tasks/", {
      method: "POST",
      token,
      body: {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        due_date: values.due_date ? new Date(values.due_date).toISOString() : null,
      },
    });

    if (values.files && values.files.length > 0) {
      for (const file of values.files) {
        try {
          await apiUpload(`/tasks/${task.id}/attachments`, file, token);
        } catch {
          showToast(`Failed to upload "${file.name}".`, "error");
        }
      }
    }

    showToast("Task created.");
    router.push("/tasks");
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">New Task</h1>
      <TaskForm submitLabel="Create task" onSubmit={handleSubmit} />
    </div>
  );
}
