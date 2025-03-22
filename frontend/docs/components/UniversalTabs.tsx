import React from 'react';
import { Tabs } from 'nextra/components';
import { useLanguage } from '../context/LanguageContext';

interface UniversalTabsProps {
  items: string[];
  children: React.ReactNode;
  optionKey?: string;
}

export const UniversalTabs: React.FC<UniversalTabsProps> = ({ 
  items, 
  children, 
  optionKey = "language"
}) => {
  const { 
    selectedLanguage, 
    setSelectedLanguage, 
    getSelectedOption, 
    setSelectedOption 
  } = useLanguage();

  const selectedValue = optionKey === "language" 
    ? selectedLanguage 
    : getSelectedOption(optionKey);
  
  const handleChange = (index: number) => {
    if (optionKey === "language") {
      setSelectedLanguage(items[index]);
    } else {
      setSelectedOption(optionKey, items[index]);
    }
  };

  return (
    <Tabs
      items={items}
      selectedIndex={items.includes(selectedValue) ? items.indexOf(selectedValue) : 0}
      onChange={handleChange}
    >
      {children}
    </Tabs>
  );
};

export default UniversalTabs;
