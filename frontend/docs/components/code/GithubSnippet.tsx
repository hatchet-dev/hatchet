import { useData } from "nextra/data";
import { CodeBlock } from "./CodeBlock";
import { RepoProps, Src } from "./codeData";

interface GithubSnippetProps {
  src: RepoProps;
  target: string;
}

type Content = {
  rawUrl: string;
};

export const GithubSnippet = ({ src, target }: GithubSnippetProps) => {
  const { contents } = useData();
  const snippet = contents.find((c: Content) =>
    c.rawUrl.endsWith(src.path)
  ) as Src;

  if (!snippet) {
    return null;
  }

  return (
    <CodeBlock
      source={{
        ...src,
        ...snippet,
      }}
      target={target}
    />
  );
};
