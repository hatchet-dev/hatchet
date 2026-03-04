import React, { createContext, useContext, useState, useRef, ReactNode, useEffect } from "react";
import { DEFAULT_LANGUAGE } from "@/lib/docs-languages";

type OptionsState = {
  [key: string]: string;
};

type LanguageContextType = {
  selectedLanguage: string;
  setSelectedLanguage: (language: string) => void;
  getSelectedOption: (key: string) => string;
  setSelectedOption: (key: string, value: string) => void;
};

const LanguageContext = createContext<LanguageContextType>({
  selectedLanguage: DEFAULT_LANGUAGE,
  setSelectedLanguage: () => {},
  getSelectedOption: () => "",
  setSelectedOption: () => {},
});

export const useLanguage = () => useContext(LanguageContext);

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [options, setOptions] = useState<OptionsState>({
    language: DEFAULT_LANGUAGE,
  });
  const dirty = useRef(false);

  const selectedLanguage = options.language || DEFAULT_LANGUAGE;

  const setSelectedLanguage = (language: string) => {
    dirty.current = true;
    setOptions((prev) => ({ ...prev, language }));
  };

  const getSelectedOption = (key: string) => {
    return options[key] || "";
  };

  const setSelectedOption = (key: string, value: string) => {
    dirty.current = true;
    setOptions((prev) => ({ ...prev, [key]: value }));
  };

  useEffect(() => {
    if (typeof window === "undefined") return;
    const savedOptions = localStorage.getItem("uiOptions");
    if (savedOptions) {
      try {
        setOptions(JSON.parse(savedOptions));
      } catch {
        const savedLanguage = localStorage.getItem("selectedLanguage");
        if (savedLanguage) {
          setOptions({ language: savedLanguage });
        }
      }
    } else {
      const savedLanguage = localStorage.getItem("selectedLanguage");
      if (savedLanguage) {
        setOptions({ language: savedLanguage });
      }
    }
  }, []);

  useEffect(() => {
    if (typeof window === "undefined" || !dirty.current) return;
    localStorage.setItem("uiOptions", JSON.stringify(options));
    localStorage.setItem(
      "selectedLanguage",
      options.language || DEFAULT_LANGUAGE
    );
  }, [options]);

  return (
    <LanguageContext.Provider
      value={{
        selectedLanguage,
        setSelectedLanguage,
        getSelectedOption,
        setSelectedOption
      }}
    >
      {children}
    </LanguageContext.Provider>
  );
};
