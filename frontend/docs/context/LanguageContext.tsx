import React, { createContext, useContext, useState, ReactNode, useEffect } from "react";

import { BiLogoGoLang, BiLogoPython, BiLogoTypescript } from 'react-icons/bi';

type LanguageType = {
  name: string;
  icon: ReactNode;
}

const LANGUAGES: LanguageType[] = [
  { name: "Python", icon: <BiLogoPython /> },
  { name: "Typescript", icon: <BiLogoTypescript /> },
  { name: "Go", icon: <BiLogoGoLang /> },
];

type LanguageContextType = {
  selectedLanguage: string;
  setSelectedLanguage: (language: string) => void;
  languages: LanguageType[];
};

const LanguageContext = createContext<LanguageContextType>({
  selectedLanguage: "Python",
  setSelectedLanguage: () => {},
  languages: [],
});

export const useLanguage = () => useContext(LanguageContext);

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [selectedLanguage, setSelectedLanguage] = useState<string>("Python");

  const languages = LANGUAGES;

  useEffect(() => {
    if (typeof window !== "undefined") {
      const savedLanguage = localStorage.getItem("selectedLanguage");
      if (savedLanguage) {
        setSelectedLanguage(savedLanguage);
      }
    }
  }, []);

  useEffect(() => {
    if (typeof window !== "undefined") {
      localStorage.setItem("selectedLanguage", selectedLanguage);
    }
  }, [selectedLanguage]);

  return (
    <LanguageContext.Provider value={{ selectedLanguage, setSelectedLanguage, languages }}>
      {children}
    </LanguageContext.Provider>
  );
};
