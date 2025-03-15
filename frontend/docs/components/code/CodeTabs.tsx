import { Children, isValidElement, useMemo } from "react";
import UniversalTabs from "../UniversalTabs";

const languages = ["Python", "Typescript", "Go"];

type CodeSource = {
  path?: string;
};

type GitHubIssue = {
  issueUrl: string;
};

type Src = CodeSource | GitHubIssue | undefined;

interface CodeTabsProps {
  children: React.ReactNode;
  src?: {
    [key: string]: Src;
  };
}

export const CodeTabs: React.FC<CodeTabsProps> = ({
  children,
}: CodeTabsProps) => {
  // Convert children to a dictionary keyed by language
  const childrenDict = useMemo(() => {
    const dict: { [key: string]: React.ReactNode } = {};
    Children.forEach(children, (child: React.ReactNode) => {
      if (isValidElement(child)) {
        dict[child.props.title] = child;
      }
    });
    return dict;
  }, [children]);

  // Create ordered array based on languages order
  const orderedChildren = useMemo(() => {
    return languages.map((lang) => childrenDict[lang]).filter(Boolean);
  }, [childrenDict]);

  return (
    <>
      <UniversalTabs items={languages}>{orderedChildren}</UniversalTabs>
    </>
  );
};

export default UniversalTabs;
