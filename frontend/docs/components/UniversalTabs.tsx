import React from "react";
import { Callout, Tabs } from "nextra/components";
import { useLanguage } from "../context/LanguageContext";

const EARLY_ACCESS_SDKS = ["Ruby"];

const EarlyAccessCallout: React.FC<{ language: string }> = ({ language }) => (
  <Callout type="info">
    <span className="text-sm">
      The {language} SDK is in early access, and may change. We&apos;d love
      your{" "}
      <a
        href="https://github.com/hatchet-dev/hatchet/issues"
        target="_blank"
        rel="noopener noreferrer"
        className="underline"
      >
        feedback
      </a>
      !
    </span>
  </Callout>
);

interface UniversalTabsProps {
  items: string[];
  children: React.ReactNode;
  optionKey?: string;
}

export const UniversalTabs: React.FC<UniversalTabsProps> = ({
  items,
  children,
  optionKey = "language",
}) => {
  const {
    selectedLanguage,
    setSelectedLanguage,
    getSelectedOption,
    setSelectedOption,
  } = useLanguage();

  const selectedValue =
    optionKey === "language" ? selectedLanguage : getSelectedOption(optionKey);

  const handleChange = (index: number) => {
    if (optionKey === "language") {
      setSelectedLanguage(items[index]);
    } else {
      setSelectedOption(optionKey, items[index]);
    }
  };

  // Inject early access callout into SDK tabs that are in early access
  const processedChildren =
    optionKey === "language"
      ? React.Children.map(children, (child) => {
          if (
            React.isValidElement<{ title?: string; children?: React.ReactNode }>(
              child,
            ) &&
            child.props.title &&
            EARLY_ACCESS_SDKS.includes(child.props.title)
          ) {
            return React.cloneElement(child, {
              children: (
                <>
                  <EarlyAccessCallout language={child.props.title} />
                  {child.props.children}
                </>
              ),
            });
          }
          return child;
        })
      : children;

  return (
    <Tabs
      items={items}
      selectedIndex={
        items.includes(selectedValue) ? items.indexOf(selectedValue) : 0
      }
      onChange={handleChange}
    >
      {processedChildren}
    </Tabs>
  );
};

export default UniversalTabs;
