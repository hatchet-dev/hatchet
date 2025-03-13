import {
  createContext,
  useContext,
  useState,
  ReactNode,
  useEffect,
} from "react";

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
}: {
  children: ReactNode;
}) => {
  const [selectedLanguage, setSelectedLanguage] = useState<string>("Python");

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
    <LanguageContext.Provider value={{ selectedLanguage, setSelectedLanguage }}>
      {children}
    </LanguageContext.Provider>
  );
};
