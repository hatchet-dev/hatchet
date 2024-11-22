import React, { createContext, useContext, useState, ReactNode, useEffect } from "react";

import { BiLogoGoLang, BiLogoPython, BiLogoTypescript } from 'react-icons/bi';
import { SiFastapi, SiExpress, SiDjango, SiFlask, SiNestjs, SiBun } from 'react-icons/si';
import { RiNextjsFill } from 'react-icons/ri';

export type PersistenceKeys = "worker" | "api-server";

type BaseLanguageType = {
  name: string;
  icon: ReactNode;
}

type LanguageType = BaseLanguageType;

const LANGUAGES: LanguageType[] = [
  { name: "Python", icon: <BiLogoPython /> },
  { name: "Typescript", icon: <BiLogoTypescript /> },
  { name: "Go", icon: <BiLogoGoLang /> },
];

const FRAMEWORKS: LanguageType[] = [
  { name: "FastAPI", icon: <SiFastapi /> },
  { name: "Express", icon: <SiExpress /> },
  { name: "Django", icon: <SiDjango /> },
  { name: "Flask", icon: <SiFlask /> },
  { name: "NestJS", icon: <SiNestjs /> },
  { name: "NextJS (App Router)", icon: <RiNextjsFill /> },
  { name: "NextJS (Page Router)", icon: <RiNextjsFill /> },
  { name: "Django REST Framework", icon: <SiDjango /> },
  { name: "Bun", icon: <SiBun /> },
  ...LANGUAGES,
];

type LanguageContextType = {
  selected: Record<PersistenceKeys, LanguageType | undefined>;
  setSelectedLanguage: (key: PersistenceKeys, language: LanguageType) => void;
  languages: Record<PersistenceKeys, LanguageType[]>;
};

const LanguageContext = createContext<LanguageContextType>({
  selected: {
    "worker": undefined,
    "api-server": undefined,
  },
  setSelectedLanguage: () => {},
  languages: {
    worker: [],
    "api-server": [],
  },
});

export const useLanguage = () => useContext(LanguageContext);

type StoredLanguageState = Record<PersistenceKeys, string | undefined>;

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [selectedLanguage, setSelectedLanguage] = useState<LanguageContextType['selected']>({
    "worker": undefined,
    "api-server": undefined,
  });

  const languages = {
    worker: LANGUAGES,
    "api-server": FRAMEWORKS,
  };

  useEffect(() => {
    if (typeof window !== "undefined") {
      const savedLanguage = localStorage.getItem("selectedLanguage");
      if (savedLanguage) {
        try {
          const storedState = JSON.parse(savedLanguage) as StoredLanguageState;
          const restoredState: LanguageContextType['selected'] = {
            worker: storedState.worker ? LANGUAGES.find(lang => lang.name === storedState.worker) : undefined,
            "api-server": storedState["api-server"] ? FRAMEWORKS.find(lang => lang.name === storedState["api-server"]) : undefined,
          };
          setSelectedLanguage(restoredState);
        } catch (e) {
          setSelectedLanguage({
            "worker": undefined,
            "api-server": undefined,
          });
          console.error(e);
        }
      }
    }
  }, []);

  useEffect(() => {
    if (typeof window !== "undefined") {
      const stateToStore: StoredLanguageState = {
        worker: selectedLanguage.worker?.name,
        "api-server": selectedLanguage["api-server"]?.name,
      };
      localStorage.setItem("selectedLanguage", JSON.stringify(stateToStore));
    }
  }, [selectedLanguage]);

  const _setSelectedLanguage = (key: PersistenceKeys, language: LanguageType) => {
    setSelectedLanguage((prev) => ({
      ...prev,
      [key]: language,
    }));
  };


  return (
    <LanguageContext.Provider value={{ selected: selectedLanguage, setSelectedLanguage: _setSelectedLanguage, languages }}>
      {children}
    </LanguageContext.Provider>
  );
};
