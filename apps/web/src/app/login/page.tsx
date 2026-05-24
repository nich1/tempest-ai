"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { api } from "@/lib/api";
import { AuthForm } from "../signup/page";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api.login(email, password);
      router.push("/jobs");
    } catch (err) {
      setError(err instanceof Error ? err.message : "login failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthForm
      title="Log in"
      submit="Log in"
      onSubmit={onSubmit}
      email={email}
      password={password}
      setEmail={setEmail}
      setPassword={setPassword}
      error={error}
      submitting={submitting}
      footer={
        <p className="text-sm text-[var(--muted)]">
          New here? <a href="/signup">Create an account</a>
        </p>
      }
    />
  );
}
