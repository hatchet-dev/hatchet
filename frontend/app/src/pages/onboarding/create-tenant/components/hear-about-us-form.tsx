import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Search,
  FileText,
  Users,
  Github,
  Calendar,
  HelpCircle,
} from 'lucide-react';
import { OnboardingStepProps } from '../types';
import { FaHackerNews, FaLinkedin, FaTwitter } from 'react-icons/fa';

interface HearAboutUsFormProps extends OnboardingStepProps<string | string[]> {}

export function HearAboutUsForm({
  value,
  onChange,
  onNext,
}: HearAboutUsFormProps) {
  const options = [
    { value: 'hackernews', label: 'Hacker News', icon: FaHackerNews },
    { value: 'search', label: 'Search Engine', icon: Search },
    { value: 'linkedin', label: 'LinkedIn', icon: FaLinkedin },
    { value: 'twitter', label: 'Twitter', icon: FaTwitter },
    { value: 'blog', label: 'Blog/Article', icon: FileText },
    { value: 'colleague', label: 'Word of Mouth', icon: Users },
    { value: 'github', label: 'GitHub', icon: Github },
    { value: 'conference', label: 'Event', icon: Calendar },
    { value: 'other', label: 'Other', icon: HelpCircle },
  ];

  // Convert value to array if it's a string for backward compatibility
  const selectedValues = Array.isArray(value) ? value : value ? [value] : [];

  // Check if "other" is selected and extract the custom value
  const otherSelection = selectedValues.find((v) => v.startsWith('other'));
  const isOtherSelected = !!otherSelection;
  const otherValue = otherSelection
    ? otherSelection.replace('other: ', '')
    : '';

  const handleOptionToggle = (optionValue: string) => {
    if (optionValue === 'other') {
      if (isOtherSelected) {
        // Remove other option
        const newValues = selectedValues.filter((v) => !v.startsWith('other'));
        onChange(newValues);
      } else {
        // Add other option with empty value
        onChange([
          ...selectedValues.filter((v) => !v.startsWith('other')),
          'other: ',
        ]);
      }
    } else {
      // Toggle regular option
      if (selectedValues.includes(optionValue)) {
        onChange(selectedValues.filter((v) => v !== optionValue));
      } else {
        onChange([
          ...selectedValues.filter((v) => !v.startsWith('other')),
          optionValue,
          ...(isOtherSelected ? [otherSelection] : []),
        ]);
      }
    }
  };

  const handleOtherTextChange = (text: string) => {
    const newValues = selectedValues.filter((v) => !v.startsWith('other'));
    onChange([...newValues, `other: ${text}`]);
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
          <Label htmlFor="other-hear-about">Please specify:</Label>
          <Textarea
            id="other-hear-about"
            placeholder="Tell us how you heard about Hatchet..."
            className="mt-2"
            value={otherValue}
            onChange={(e) => handleOtherTextChange(e.target.value)}
          />
        </div>
      )}

      <div className="mt-6">
        <Button
          onClick={onNext}
          className="w-full"
          disabled={selectedValues.length === 0}
        >
          Continue
        </Button>
      </div>
    </div>
  );
}
