import React from 'react';
import { Tabs } from 'nextra/components';
import { PersistenceKeys, useLanguage } from '../context/LanguageContext';

interface UniversalTabsProps {
  children: React.ReactNode;
  key?: PersistenceKeys;
}

export const UniversalTabs: React.FC<UniversalTabsProps> = ({ children, key = "worker" }) => {
  const { selected, setSelectedLanguage, languages } = useLanguage();
  
  const selectedIndex = selected[key] 
    ? languages[key].findIndex(lang => lang.name === selected[key]?.name)
    : 0;

  return (
    <Tabs
      items={languages[key].map(lang => lang.name)}
      selectedIndex={selectedIndex}
      onChange={(index) => setSelectedLanguage(key, languages[key][index])}
    >
      {children}
    </Tabs>
  );
};

export default UniversalTabs;
