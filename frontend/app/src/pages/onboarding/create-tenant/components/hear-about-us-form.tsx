import { OnboardingStepProps } from '../types';
import {
  extractOtherSelection,
  toggleOtherOption,
  toggleRegularOption,
  updateOtherText,
} from '../utils/other-selection';
import { Button } from '@/components/v1/ui/button';
import { Card, CardContent } from '@/components/v1/ui/card';
import { Label } from '@/components/v1/ui/label';
import { Textarea } from '@/components/v1/ui/textarea';
import {
  Search,
  FileText,
  Users,
  Github,
  Calendar,
  HelpCircle,
} from 'lucide-react';
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
                  <Icon className="h-5 w-5 flex-shrink-0 text-gray-600 dark:text-gray-400" />
                  <span className="text-sm font-medium">{option.label}</span>
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
