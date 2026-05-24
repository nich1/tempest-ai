"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import type { Job } from "@/lib/types";

export default function JobDetailPage() {
  const params = useParams<{ id: string }>();
  const [job, setJob] = useState<Job | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!params?.id) return;
    let cancelled = false;
    const tick = async () => {
      try {
        const j = await api.getJob(params.id);
        if (!cancelled) setJob(j);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "load failed");
        }
      }
    };
    tick();
    const t = setInterval(tick, 2000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [params?.id]);

  if (error) return <div className="text-red-400">{error}</div>;
  if (!job) return <div className="text-[var(--muted)]">loading...</div>;

  return (
    <div className="space-y-6">
      <header>
        <div className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)]">
          job
        </div>
        <h1 className="mt-2 text-2xl font-mono tracking-tight">{job.id.slice(0, 8)}</h1>
        <div className="mt-1 text-xs font-mono text-[var(--muted)]">
          {job.status} &middot; attempt {job.attempt} &middot; {job.provider}
        </div>
      </header>

      <Section title="Prompt">
        <pre className="whitespace-pre-wrap font-mono text-sm">{job.prompt}</pre>
      </Section>

      <Section title="Inputs">
        <pre className="whitespace-pre-wrap font-mono text-xs">
          {JSON.stringify(job.inputs, null, 2)}
        </pre>
      </Section>

      {job.output ? (
        <Section title="Output">
          <pre className="whitespace-pre-wrap font-mono text-xs">
            {JSON.stringify(job.output, null, 2)}
          </pre>
        </Section>
      ) : null}

      {job.error_message ? (
        <Section title="Error">
          <pre className="whitespace-pre-wrap font-mono text-xs text-red-400">
            {job.error_message}
          </pre>
        </Section>
      ) : null}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-sm border border-[var(--border)] bg-[var(--card)] p-5">
      <h2 className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)] mb-3">
        {title}
      </h2>
      {children}
    </section>
  );
}
