"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { api } from "@/lib/api";

export default function SignupPage() {
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
      await api.signup(email, password);
      router.push("/jobs");
    } catch (err) {
      setError(err instanceof Error ? err.message : "signup failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthForm
      title="Create an account"
      submit="Sign up"
      onSubmit={onSubmit}
      email={email}
      password={password}
      setEmail={setEmail}
      setPassword={setPassword}
      error={error}
      submitting={submitting}
      footer={
        <p className="text-sm text-[var(--muted)]">
          Already have an account? <a href="/login">Log in</a>
        </p>
      }
    />
  );
}

type AuthFormProps = {
  title: string;
  submit: string;
  onSubmit: (e: React.FormEvent) => void;
  email: string;
  password: string;
  setEmail: (s: string) => void;
  setPassword: (s: string) => void;
  error: string | null;
  submitting: boolean;
  footer?: React.ReactNode;
};

export function AuthForm(props: AuthFormProps) {
  return (
    <div className="max-w-md mx-auto rounded-sm border border-[var(--border)] bg-[var(--card)] p-6">
      <h1 className="text-xl font-semibold tracking-tight">{props.title}</h1>
      <form onSubmit={props.onSubmit} className="mt-6 space-y-5">
        <Field
          label="Email"
          type="email"
          value={props.email}
          onChange={props.setEmail}
        />
        <Field
          label="Password"
          type="password"
          value={props.password}
          onChange={props.setPassword}
        />
        {props.error ? (
          <div className="text-sm text-red-400 font-mono">{props.error}</div>
        ) : null}
        <button
          type="submit"
          disabled={props.submitting}
          className="w-full rounded-sm bg-[var(--accent)] px-4 py-2 text-white text-sm font-semibold uppercase tracking-[0.12em] disabled:opacity-50"
        >
          {props.submitting ? "..." : props.submit}
        </button>
        {props.footer}
      </form>
    </div>
  );
}

export function Field({
  label,
  type,
  value,
  onChange,
}: {
  label: string;
  type: string;
  value: string;
  onChange: (s: string) => void;
}) {
  return (
    <label className="block">
      <span className="block text-[10px] font-mono uppercase tracking-[0.18em] text-[var(--muted)] mb-1.5">
        {label}
      </span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-sm border border-[var(--border)] bg-[var(--bg)] px-3 py-2 text-[var(--fg)] focus:outline-none focus:border-[var(--accent)]"
        required
      />
    </label>
  );
}
