import React from "react";
import { DOC_LANGUAGES } from "@/lib/docs-languages";
import UniversalTabs from "../UniversalTabs";

const languages = [...DOC_LANGUAGES];

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

export const CodeTabs: React.FC<CodeTabsProps> = ({ children }) => {
  // Convert children to a dictionary keyed by language
  const childrenDict = React.useMemo(() => {
    const dict: { [key: string]: React.ReactNode } = {};
    React.Children.forEach(children, (child) => {
      if (React.isValidElement(child)) {
        dict[child.props.title] = child;
      }
    });
    return dict;
  }, [children]);

  // Create ordered array based on languages order
  const orderedChildren = React.useMemo(() => {
    return languages.map((lang) => childrenDict[lang]).filter(Boolean);
  }, [childrenDict]);

  return (
    <>
      <UniversalTabs items={languages}>{orderedChildren}</UniversalTabs>
    </>
  );
};

export default UniversalTabs;
