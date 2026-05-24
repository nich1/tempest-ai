import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";

const sans = Inter({
  subsets: ["latin"],
  variable: "--font-sans",
  display: "swap",
});

const mono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Tempest AI",
  description: "LLM schema manager - distributed Go + Next.js demo",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={`${sans.variable} ${mono.variable}`}>
      <body className="min-h-screen font-sans antialiased">
        <header className="border-b border-[var(--border)] bg-[var(--card)]">
          <div className="mx-auto max-w-5xl px-6 py-4 flex items-center justify-between">
            <a
              href="/"
              className="font-mono text-sm tracking-tight text-[var(--fg)] no-underline"
            >
              <span className="text-[var(--muted)]">$ </span>tempest
              <span className="text-[var(--accent)]">.ai</span>
            </a>
            <nav className="flex items-center gap-6 text-xs uppercase tracking-[0.18em] text-[var(--muted)]">
              <a href="/jobs" className="hover:text-[var(--fg)] no-underline">jobs</a>
              <a href="/login" className="hover:text-[var(--fg)] no-underline">log in</a>
              <a href="/signup" className="hover:text-[var(--fg)] no-underline">sign up</a>
            </nav>
          </div>
        </header>
        <main className="mx-auto max-w-5xl px-6 py-10">{children}</main>
      </body>
    </html>
  );
}
