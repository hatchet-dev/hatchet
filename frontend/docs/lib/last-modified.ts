import { execFileSync } from "node:child_process";
import path from "node:path";
import fs from "node:fs";

const CONTENT_DIR = path.join(process.cwd(), "content/docs");
const cache = new Map<string, number | null>();

export function getLastModified(slug: string[]): number | null {
  const key = slug.join("/");
  if (cache.has(key)) return cache.get(key) ?? null;

  const base = path.join(CONTENT_DIR, ...slug);
  const file = fs.existsSync(`${base}.mdx`)
    ? `${base}.mdx`
    : path.join(base, "index.mdx");

  let ts: number | null = null;
  try {
    const out = execFileSync("git", ["log", "-1", "--format=%cI", "--", file], {
      encoding: "utf-8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
    if (out) ts = new Date(out).getTime();
  } catch {
    ts = null;
  }
  if (ts === null && fs.existsSync(file)) {
    ts = fs.statSync(file).mtimeMs;
  }

  cache.set(key, ts);
  return ts;
}
