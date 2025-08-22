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

interface HearAboutUsFormProps extends OnboardingStepProps<string> {}

export function HearAboutUsForm({
  value,
  onChange,
  onNext,
}: HearAboutUsFormProps) {
  const options = [
    { value: 'search', label: 'Search Engine', icon: Search },
    { value: 'hackernews', label: 'Hacker News', icon: FaHackerNews },
    { value: 'linkedin', label: 'LinkedIn', icon: FaLinkedin },
    { value: 'twitter', label: 'Twitter', icon: FaTwitter },
    { value: 'blog', label: 'Blog/Article', icon: FileText },
    { value: 'colleague', label: 'Word of Mouth', icon: Users },
    { value: 'github', label: 'GitHub', icon: Github },
    { value: 'conference', label: 'Event', icon: Calendar },
    { value: 'other', label: 'Other', icon: HelpCircle },
  ];

  const isOther = value.startsWith('other');
  const otherValue = isOther ? value.replace('other: ', '') : '';

  const handleCardClick = (selectedValue: string) => {
    if (selectedValue === 'other') {
      onChange('other: ');
    } else {
      onChange(selectedValue);
      onNext?.();
    }
  };

  return (
    <>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-3 gap-4">
        {options.map((option) => {
          const Icon = option.icon;
          const isSelected =
            (isOther && option.value === 'other') ||
            (!isOther && value === option.value);

          return (
            <Card
              key={option.value}
              onClick={() => handleCardClick(option.value)}
              className={`relative w-full before:content-[''] before:block before:pb-[100%] cursor-pointer transition-all hover:shadow-lg ${
                isSelected
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-950'
                  : 'hover:border-gray-300 dark:hover:border-gray-600'
              }`}
            >
              <CardContent className="absolute inset-0 p-4 flex flex-col items-center justify-center text-center space-y-2">
                <Icon
                  className={`w-8 h-8 ${isSelected ? 'text-blue-600 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'}`}
                />
                <div className="font-medium text-sm">{option.label}</div>
              </CardContent>
            </Card>
          );
        })}

        {isOther && (
          <div className="col-span-full mt-6 space-y-3">
            <Label htmlFor="other-hear-about">Please specify:</Label>
            <Textarea
              id="other-hear-about"
              placeholder="Tell us how you heard about Hatchet..."
              className="mt-2"
              value={otherValue}
              onChange={(e) => onChange(`other: ${e.target.value}`)}
            />
            <Button onClick={onNext} className="w-full">
              Continue
            </Button>
          </div>
        )}
      </div>
    </>
  );
}
