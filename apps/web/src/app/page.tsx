export default function HomePage() {
  return (
    <div className="space-y-10">
      <section>
        <div className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)]">
          README.md
        </div>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">Tempest AI</h1>
        <p className="mt-3 max-w-2xl text-[var(--muted)] leading-relaxed">
          A distributed LLM schema manager. Define typed input/output schemas, submit
          jobs, and let pluggable LLM providers fill in the answer&nbsp;&mdash;
          asynchronously, across horizontally-scaled Go consumers.
        </p>
      </section>

      <section className="grid gap-3 md:grid-cols-3">
        <Card
          tag="api"
          title="API server"
          body="Gin-based HTTP server with cookie auth, presigned MinIO uploads, Asynq enqueue."
        />
        <Card
          tag="consumers"
          title="Workers"
          body="Asynq workers running JSON-mode LLM calls against Ollama, OpenAI, Anthropic, or Gemini."
        />
        <Card
          tag="storage"
          title="Persistence"
          body="Postgres for metadata, Redis for the queue, MinIO for files - all S3-compatible in prod."
        />
      </section>

      <section>
        <h2 className="text-xs font-mono uppercase tracking-[0.18em] text-[var(--muted)]">
          Get started
        </h2>
        <ol className="mt-3 list-decimal pl-6 space-y-1 text-[var(--muted)]">
          <li>
            <a href="/signup">Create an account</a>
          </li>
          <li>Define an input/output schema, prompt, and (optionally) attach a file</li>
          <li>
            Watch the job move PENDING &rarr; PROCESSING &rarr; COMPLETED in
            <a href="/jobs"> /jobs</a>
          </li>
        </ol>
      </section>
    </div>
  );
}

function Card({ tag, title, body }: { tag: string; title: string; body: string }) {
  return (
    <div className="rounded-sm border border-[var(--border)] bg-[var(--card)] p-5">
      <div className="text-[10px] font-mono uppercase tracking-[0.18em] text-[var(--muted)]">
        {tag}
      </div>
      <h3 className="mt-1 font-semibold tracking-tight">{title}</h3>
      <p className="mt-2 text-sm text-[var(--muted)] leading-relaxed">{body}</p>
    </div>
  );
}
