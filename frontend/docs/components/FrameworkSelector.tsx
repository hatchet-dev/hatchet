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
  const { selectedLanguage, setSelectedLanguage, languages } = useLanguage();


  return (
    <div>
        <span className="text-sm font-medium mb-2 inline-block">Choose your language:</span><br />
        <Select 
            value={selectedLanguage}
            onValueChange={(value) => setSelectedLanguage(value)}
        >
            <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Language" />
            </SelectTrigger>
            <SelectContent>
                {languages.map(({ name, icon }) => (
                    <SelectItem key={name} value={name}>
                        <span className="flex items-center gap-2">
                            {icon} {name}
                        </span>
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    </div>
  );
};

export default FrameworkSelector;
