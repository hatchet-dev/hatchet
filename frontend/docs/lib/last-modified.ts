import { execFileSync } from "node:child_process";
import path from "node:path";
import fs from "node:fs";

const CONTENT_DIR = path.join(process.cwd(), "content/docs");
const cache = new Map<string, number | null>();

/** Git commit time (ms) of a doc page, resolved from its slug. Cached per build. */
export function getLastModified(slug: string[]): number | null {
  const key = slug.join("/");
  if (cache.has(key)) return cache.get(key) ?? null;

  const base = path.join(CONTENT_DIR, ...slug);
  const file = fs.existsSync(`${base}.mdx`)
    ? `${base}.mdx`
    : path.join(base, "index.mdx");

  let ts: number | null = null;
  try {
    const out = execFileSync(
      "git",
      ["log", "-1", "--format=%cI", "--", file],
      { encoding: "utf-8", stdio: ["ignore", "pipe", "ignore"] },
    ).trim();
    if (out) ts = new Date(out).getTime();
  } catch {
    ts = null;
  }
  // fall back to file mtime when git has no record (fresh/untracked files)
  if (ts === null && fs.existsSync(file)) {
    ts = fs.statSync(file).mtimeMs;
  }

  cache.set(key, ts);
  return ts;
}
