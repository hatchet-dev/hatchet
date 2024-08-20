import React, { createContext, useContext, useState, ReactNode } from "react";

type LanguageContextType = {
  selectedLanguage: string;
  setSelectedLanguage: (language: string) => void;
};

const LanguageContext = createContext<LanguageContextType>({
  selectedLanguage: "Python",
  setSelectedLanguage: () => {},
});

export const useLanguage = () => useContext(LanguageContext);

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [selectedLanguage, setSelectedLanguage] = useState("Python");
  return (
    <LanguageContext.Provider value={{ selectedLanguage, setSelectedLanguage }}>
      {children}
    </LanguageContext.Provider>
  );
};
