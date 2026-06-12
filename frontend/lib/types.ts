export type TaskStatus = "todo" | "in_progress" | "done";
export type TaskPriority = "low" | "medium" | "high";

export interface Task {
  id: string;
  user_id: string;
  user_email?: string;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  due_date: string | null;
  created_at: string;
  updated_at: string;
}

export interface TaskListResponse {
  data: Task[];
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface User {
  id: string;
  email: string;
  role: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface TaskAttachment {
  id: string;
  task_id: string;
  user_id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface TaskActivity {
  id: string;
  task_id: string;
  user_id: string;
  user_email?: string;
  action: string;
  details: string;
  created_at: string;
}

export interface ApiErrorBody {
  error: string;
  details?: Record<string, string>;
}
