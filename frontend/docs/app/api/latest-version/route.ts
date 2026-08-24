const REPOS: Record<string, string> = {
  hatchet: "hatchet-dev/hatchet",
  "hatchet-embedded": "hatchet-dev/hatchet-embedded",
};

export async function GET(request: Request) {
  const repoKey = new URL(request.url).searchParams.get("repo") ?? "hatchet";
  const repo = REPOS[repoKey];

  if (!repo) {
    return Response.json({ tag_name: null }, { status: 400 });
  }

  try {
    const res = await fetch(
      `https://api.github.com/repos/${repo}/releases/latest`,
      { next: { revalidate: 3600 } },
    );
    if (!res.ok) {
      throw new Error(`GitHub API returned ${res.status}`);
    }
    const { tag_name } = await res.json();
    if (!/^v\d+\.\d+\.\d+(-\d+)?$/.test(tag_name)) {
      throw new Error(`unexpected tag_name: ${tag_name}`);
    }
    return Response.json(
      { tag_name },
      {
        headers: {
          "Cache-Control": "public, max-age=3600, stale-while-revalidate=3600",
        },
      },
    );
  } catch {
    return Response.json({ tag_name: null }, { status: 502 });
  }
}
