"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Job } from "@/lib/types";

const SAMPLE_REQUEST = {
  input_schema: [
    { name: "company", type: "string", required: true },
    { name: "country", type: "string", required: true },
  ],
  output_schema: [
    { name: "founded_year", type: "integer", required: true },
    { name: "industries", type: "array", required: true, items: { name: "i", type: "string" } },
  ],
  inputs: { company: "Anthropic", country: "USA" },
  prompt: "Return the founded year and primary industries.",
  system_prompt: "You are a precise extraction assistant.",
};

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [body, setBody] = useState(JSON.stringify(SAMPLE_REQUEST, null, 2));
  const [submitting, setSubmitting] = useState(false);

  const refresh = async () => {
    try {
      const resp = await api.listJobs();
      setJobs(resp.jobs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to load");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
    const t = setInterval(refresh, 3000);
    return () => clearInterval(t);
  }, []);

  const submit = async () => {
    setError(null);
    setSubmitting(true);
    try {
      const parsed = JSON.parse(body);
      await api.createJob(parsed);
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "submit failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-10">
      <section>
        <div className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)]">
          /jobs
        </div>
        <h1 className="mt-2 text-2xl font-semibold tracking-tight">Jobs</h1>
        <p className="text-sm text-[var(--muted)] mt-2">
          Polls every 3 seconds. Status flips PENDING &rarr; PROCESSING &rarr; COMPLETED as the
          consumer picks them up.
        </p>
      </section>

      <section className="rounded-sm border border-[var(--border)] bg-[var(--card)] p-5">
        <h2 className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)] mb-3">
          Submit a job
        </h2>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={14}
          className="w-full font-mono text-sm rounded-sm border border-[var(--border)] bg-[var(--bg)] p-3 text-[var(--fg)] focus:outline-none focus:border-[var(--accent)]"
          spellCheck={false}
        />
        {error ? (
          <div className="mt-2 text-sm font-mono text-red-400">{error}</div>
        ) : null}
        <button
          onClick={submit}
          disabled={submitting}
          className="mt-3 rounded-sm bg-[var(--accent)] px-4 py-2 text-white text-sm font-semibold uppercase tracking-[0.12em] disabled:opacity-50"
        >
          {submitting ? "..." : "Submit job"}
        </button>
      </section>

      <section>
        <h2 className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)] mb-3">
          Recent jobs
        </h2>
        {loading ? (
          <div className="text-sm text-[var(--muted)]">loading...</div>
        ) : jobs.length === 0 ? (
          <div className="text-sm text-[var(--muted)]">No jobs yet.</div>
        ) : (
          <ul className="space-y-2">
            {jobs.map((j) => (
              <li
                key={j.id}
                className="rounded-sm border border-[var(--border)] bg-[var(--card)] p-4 hover:border-[var(--accent)] transition-colors"
              >
                <div className="flex items-center justify-between">
                  <a href={`/jobs/${j.id}`} className="font-mono text-sm">
                    {j.id.slice(0, 8)}
                  </a>
                  <StatusPill status={j.status} />
                </div>
                <div className="text-xs font-mono text-[var(--muted)] mt-1">
                  {j.provider} &middot; {new Date(j.created_at).toLocaleString()}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function StatusPill({ status }: { status: string }) {
  const tone =
    status === "COMPLETED"
      ? "text-emerald-400"
      : status === "FAILED"
        ? "text-red-400"
        : status === "PROCESSING"
          ? "text-[var(--accent)]"
          : "text-[var(--muted)]";
  return (
    <span className={`text-[10px] font-mono uppercase tracking-[0.18em] ${tone}`}>
      {status}
    </span>
  );
}
