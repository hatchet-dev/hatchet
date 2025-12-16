import { Label } from '@/components/v1/ui/label';
import { Textarea } from '@/components/v1/ui/textarea';
import { ReviewedButtonTemp } from '@/components/v1/ui/button';
import { Card, CardContent } from '@/components/v1/ui/card';
import {
  Database,
  Workflow,
  Brain,
  HelpCircle,
  FileText,
  Webhook,
} from 'lucide-react';
import { OnboardingStepProps } from '../types';
import {
  extractOtherSelection,
  toggleOtherOption,
  toggleRegularOption,
  updateOtherText,
} from '../utils/other-selection';

interface WhatBuildingFormProps
  extends OnboardingStepProps<string | string[]> {}

export function WhatBuildingForm({
  value,
  onChange,
  onNext,
}: WhatBuildingFormProps) {
  const options = [
    { value: 'ai-agents', label: 'AI Agents', icon: Brain },
    {
      value: 'document-ingestion',
      label: 'Document Ingestion',
      icon: FileText,
    },
    { value: 'data-pipeline', label: 'Data Pipelines', icon: Database },
    {
      value: 'internal-automations',
      label: 'Internal Automations',
      icon: Workflow,
    },
    { value: 'webhooks', label: 'Webhooks', icon: Webhook },
    { value: 'other', label: 'Other', icon: HelpCircle },
  ];

  const selectedValues = Array.isArray(value) ? value : value ? [value] : [];

  // Extract "other" selection information using helper
  const { isOtherSelected, otherValue, otherSelection } =
    extractOtherSelection(selectedValues);

  const handleOptionToggle = (optionValue: string) => {
    if (optionValue === 'other') {
      const newValues = toggleOtherOption(selectedValues, isOtherSelected);
      onChange(newValues);
    } else {
      const newValues = toggleRegularOption(
        selectedValues,
        optionValue,
        isOtherSelected,
        otherSelection,
      );
      onChange(newValues);
    }
  };

  const handleOtherTextChange = (text: string) => {
    const newValues = updateOtherText(selectedValues, text);
    onChange(newValues);
  };

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        {options.map((option) => {
          const Icon = option.icon;
          const isSelected =
            option.value === 'other'
              ? isOtherSelected
              : selectedValues.includes(option.value);

          return (
            <Card
              key={option.value}
              onClick={() => handleOptionToggle(option.value)}
              className={`cursor-pointer transition-all hover:shadow-md ${
                isSelected
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-950'
                  : 'hover:border-gray-300 dark:hover:border-gray-600'
              }`}
            >
              <CardContent className="p-4">
                <div className="flex items-center space-x-3">
                  <Icon className="w-5 h-5 text-gray-600 dark:text-gray-400 flex-shrink-0" />
                  <span className="font-medium text-sm">{option.label}</span>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {isOtherSelected && (
        <div className="mt-6 space-y-3">
          <Label htmlFor="other-description">
            Please describe what you're building:
          </Label>
          <Textarea
            id="other-description"
            placeholder="Tell us about your project..."
            className="mt-2"
            value={otherValue}
            onChange={(e) => handleOtherTextChange(e.target.value)}
          />
        </div>
      )}

      <div className="mt-6">
        <ReviewedButtonTemp
          onClick={onNext}
          className="w-full"
          disabled={selectedValues.length === 0}
        >
          Continue
        </ReviewedButtonTemp>
      </div>
    </div>
  );
}
