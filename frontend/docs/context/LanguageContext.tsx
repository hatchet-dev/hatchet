import React, { createContext, useContext, useState, ReactNode, useEffect } from "react";

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
  selectedLanguage: "Python",
  setSelectedLanguage: () => {},
  getSelectedOption: () => "",
  setSelectedOption: () => {},
});

export const useLanguage = () => useContext(LanguageContext);

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [options, setOptions] = useState<OptionsState>({ language: "Python" });

  // For backward compatibility
  const selectedLanguage = options.language || "Python";
  const setSelectedLanguage = (language: string) => {
    setOptions((prev) => ({ ...prev, language }));
  };

  const getSelectedOption = (key: string) => {
    return options[key] || "";
  };

  const setSelectedOption = (key: string, value: string) => {
    setOptions((prev) => ({ ...prev, [key]: value }));
  };

  useEffect(() => {
    if (typeof window !== "undefined") {
      // Load all saved options from localStorage
      const savedOptions = localStorage.getItem("uiOptions");
      if (savedOptions) {
        try {
          setOptions(JSON.parse(savedOptions));
        } catch (e) {
          // Fallback for backward compatibility
          const savedLanguage = localStorage.getItem("selectedLanguage");
          if (savedLanguage) {
            setOptions({ language: savedLanguage });
          }
        }
      } else {
        // Backward compatibility
        const savedLanguage = localStorage.getItem("selectedLanguage");
        if (savedLanguage) {
          setOptions({ language: savedLanguage });
        }
      }
    }
  }, []);

  useEffect(() => {
    if (typeof window !== "undefined") {
      // Save all options to localStorage
      localStorage.setItem("uiOptions", JSON.stringify(options));
      
      // Also save language separately for backward compatibility
      localStorage.setItem("selectedLanguage", options.language || "Python");
    }
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
