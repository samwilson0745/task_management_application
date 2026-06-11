"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";

export default function Home() {
  const { user, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;
    router.replace(user ? "/tasks" : "/login");
  }, [isLoading, user, router]);

  return (
    <div className="flex flex-1 items-center justify-center text-zinc-500">
      Loading...
    </div>
  );
}
