import React from 'react';
import { useLanguage } from '../context/LanguageContext';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from "@/components/ui/select"

  
interface FrameworkSelectorProps {
  className?: string;
}

export const FrameworkSelector: React.FC<FrameworkSelectorProps> = ({ className }) => {
  const { selected, setSelectedLanguage, languages } = useLanguage();

  return (
    <div>
      <span className="text-sm font-medium mb-2 inline-block">Choose your worker language:</span>
      <Select 
        value={selected.worker?.name}
        onValueChange={(value) => {
          const selectedLang = languages.worker.find(lang => lang.name === value);
          if (selectedLang) {
            setSelectedLanguage("worker", selectedLang);
          }
        }}
      >
        <SelectTrigger className="w-[180px]">
          <SelectValue placeholder="Language" />
        </SelectTrigger>
        <SelectContent>
          {languages.worker.map((lang) => (
            <SelectItem key={lang.name} value={lang.name}>
              <span className="flex items-center gap-2">
                {lang.icon} {lang.name}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <span className="text-sm font-medium mb-2 inline-block">Choose your API server:</span>
      <Select 
        value={selected['api-server']?.name}
        onValueChange={(value) => {
          const selectedLang = languages.worker.find(lang => lang.name === value);
          if (selectedLang) {
            setSelectedLanguage("worker", selectedLang);
          }
        }}
      >
        <SelectTrigger className="w-[180px]">
          <SelectValue placeholder="Language" />
        </SelectTrigger>
        <SelectContent>
          {languages.worker.map((lang) => (
            <SelectItem key={lang.name} value={lang.name}>
              <span className="flex items-center gap-2">
                {lang.icon} {lang.name}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};

export default FrameworkSelector;
