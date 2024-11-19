"use client";

import React from 'react';
import { useLanguage } from '../context/LanguageContext';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
    DialogClose,
  } from "@/components/ui/dialog"
import { Button } from '@/components/ui/button';
import { BiChevronDown, BiPlus } from 'react-icons/bi';
  
  
interface FrameworkSelectorProps {
  className?: string;
}

export const FrameworkSelector: React.FC<FrameworkSelectorProps> = ({ className }) => {
  const { selected, setSelectedLanguage, languages } = useLanguage();

  return (
    <>
      <Dialog>
        <DialogTrigger asChild>
          <Button className="inline-flex items-center gap-2 px-4 py-2 border rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 dark:border-gray-700 dark:bg-background dark:text-foreground">
            {selected.worker?.icon} {selected.worker?.name || "Select Language"}
            {selected['api-server'] && selected['api-server']?.name !== selected.worker?.name && <> <BiPlus /> {selected['api-server']?.icon} {selected['api-server']?.name}</>}
            <BiChevronDown className="ml-2" />
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-[725px] w-[95vw] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Personalize Your Docs</DialogTitle>
            <DialogDescription>
              Select the programming language and frameworks you want to use for code examples.
            </DialogDescription>
          </DialogHeader>
          
          <div className="space-y-4">
            <div className="space-y-2">
              <h4 className="font-medium">Hatchet Worker Language</h4>
              <p className="text-sm">
                The worker language is the language that your hatchet functions and workflows will be written in.
              </p>
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
                {languages.worker.map((lang) => (
                  <button
                    key={lang.name}
                    className={`
                      aspect-square flex flex-col items-center justify-center gap-1.5 
                      border rounded-md hover:bg-gray-50 transition-colors p-1
                      ${selected.worker?.name === lang.name 
                        ? 'border-blue-500 bg-blue-50 ring-1 ring-blue-500 ring-opacity-50 text-blue-900' 
                        : ''}
                    `}
                    onClick={() => setSelectedLanguage("worker", lang)}
                  >
                    <div className="text-3xl">{lang.icon}</div>
                    <div className="text-xs font-medium leading-none text-center whitespace-pre-line">
                      {lang.name.replace('(', '\n(')}
                    </div>
                  </button>
                ))}
              </div>
            </div>
            
            <div className="space-y-2">
              <h4 className="font-medium">API Server</h4>
              <p className="text-sm">
                The API server framework is where you schedule work for your hatchet functions.
              </p>
              <div className="max-h-[230px] overflow-y-auto pr-2">
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
                  {languages['api-server'].map((lang) => (
                    <button
                      key={lang.name}
                      className={`
                        aspect-square flex flex-col items-center justify-center gap-2.5 
                        border rounded-md hover:bg-gray-50 transition-colors p-1
                        ${selected['api-server']?.name === lang.name 
                          ? 'border-blue-500 bg-blue-50 ring-1 ring-blue-500 ring-opacity-50 text-blue-900' 
                          : ''}
                      `}
                      onClick={() => setSelectedLanguage("api-server", lang)}
                    >
                      <div className="text-3xl">{lang.icon}</div>
                      <div className="text-xs font-medium leading-none text-center whitespace-pre-line">
                        {lang.name.replace('(', '\n(')}
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row justify-between mt-6 pt-4 border-t gap-4">
            <p className="text-sm">
              Don't see your favorite framework?
              <a href="https://github.com/hatchet-dev/hatchet/issues/new?template=feature_request.md&labels=feature-request&title=Framework+Request:" target="_blank">
                <Button variant="link" size="sm">Request a Framework</Button>
              </a>
            </p>
            <DialogClose asChild>
              <Button variant="outline">Save Changes</Button>
            </DialogClose>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default FrameworkSelector;
