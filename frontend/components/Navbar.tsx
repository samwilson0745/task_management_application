"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";

export default function Navbar() {
  const { user, logout, isLoading } = useAuth();
  const router = useRouter();

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  return (
    <header className="border-b border-zinc-200 bg-white">
      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 h-14 flex items-center justify-between">
        <Link href="/" className="font-semibold text-lg text-zinc-900">
          Task Manager
        </Link>

        <nav className="flex items-center gap-4 text-sm">
          {!isLoading && user && (
            <>
              <Link href="/tasks" className="text-zinc-600 hover:text-zinc-900">
                Tasks
              </Link>
              <Link href="/tasks/new" className="text-zinc-600 hover:text-zinc-900">
                New Task
              </Link>
              <span className="text-zinc-400 hidden sm:inline">{user.email}</span>
              <button
                onClick={handleLogout}
                className="rounded-md bg-zinc-900 px-3 py-1.5 text-white hover:bg-zinc-700"
              >
                Log out
              </button>
            </>
          )}
          {!isLoading && !user && (
            <>
              <Link href="/login" className="text-zinc-600 hover:text-zinc-900">
                Log in
              </Link>
              <Link
                href="/signup"
                className="rounded-md bg-zinc-900 px-3 py-1.5 text-white hover:bg-zinc-700"
              >
                Sign up
              </Link>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
