const defaultUser = "hatchet-dev";

type LanguageCode = "ts" | "py" | "go";

const defaultRepos: Record<LanguageCode, string> = {
  ts: "hatchet-typescript",
  py: "hatchet-python",
  go: "hatchet",
} as const;

const localPaths: Record<LanguageCode, string> = {
  ts: "sdk-typescript",
  py: "sdk-python",
  go: "oss",
} as const;

export const extToLanguage: Record<LanguageCode, string> = {
  ts: "typescript",
  py: "python",
  go: "go",
} as const;

const defaultBranch = "main";

export type RepoProps = {
  user?: string;
  repo?: string;
  branch?: string;
  path: string;
};

const getLocalUrl = (ext: LanguageCode, { path }: RepoProps) => {
  return `http://localhost:4001/${localPaths[ext]}/${path}`;
};

const isDev = process?.env?.NODE_ENV === "development";

const getRawUrl = ({ user, repo, branch, path }: RepoProps) => {
  const ext = path.split(".").pop() as LanguageCode | undefined;

  if (!ext) {
    throw new Error(`No extension found for path: ${path}`);
  }

  if (isDev) {
    return getLocalUrl(ext, { path });
  }
  return `https://raw.githubusercontent.com/${user || defaultUser}/${repo || defaultRepos[ext]}/refs/heads/${branch || defaultBranch}/${path}`;
};

const getUIUrl = ({ user, repo, branch, path }: RepoProps) => {
  const ext = path.split(".").pop() as LanguageCode | undefined;

  if (!ext) {
    throw new Error(`No extension found for path: ${path}`);
  }

  if (isDev) {
    return getLocalUrl(ext, { path });
  }

  return `https://github.com/${user || defaultUser}/${repo || defaultRepos[ext]}/blob/${branch || defaultBranch}/${path}`;
};

export type Src = {
  raw: string;
  props: RepoProps;
  rawUrl: string;
  githubUrl: string;
  language?: string;
};
export const getSnippets = (
  props: RepoProps[]
): Promise<{ props: { ssg: { contents: Src[] } } }> => {
  return Promise.all(
    props.map(async (prop) => {
      const rawUrl = getRawUrl(prop);
      const githubUrl = getUIUrl(prop);
      const fileExt = prop.path.split(".").pop() as keyof typeof extToLanguage;
      const language = extToLanguage[fileExt];

      try {
        const response = await fetch(rawUrl);
        const raw = await response.text();

        return {
          raw,
          props: prop,
          rawUrl,
          githubUrl,
          language,
        };
      } catch (error) {
        // Return object with empty raw content but preserve URLs on failure
        return {
          raw: "",
          props: prop,
          rawUrl,
          githubUrl,
          language,
        };
      }
    })
  ).then((results) => ({
    props: {
      ssg: {
        contents: results,
      },
      // revalidate every 60 seconds
      revalidate: 60,
    },
  }));
};
