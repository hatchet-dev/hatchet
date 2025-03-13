import { Tabs } from "nextra/components";
import { useLanguage } from "../context/LanguageContext";

interface UniversalTabsProps {
  items: string[];
  children: React.ReactNode;
}

export const UniversalTabs: React.FC<UniversalTabsProps> = ({
  items,
  children,
}: UniversalTabsProps) => {
  const { selectedLanguage, setSelectedLanguage } = useLanguage();

  return (
    <Tabs
      items={items}
      selectedIndex={
        items.includes(selectedLanguage) ? items.indexOf(selectedLanguage) : 0
      }
      onChange={(index) => setSelectedLanguage(items[index])}
    >
      {children}
    </Tabs>
  );
};

export default UniversalTabs;
