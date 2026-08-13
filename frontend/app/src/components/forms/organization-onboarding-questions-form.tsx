import { Button } from '@/components/v1/ui/button';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { RadioGroup, RadioGroupCardItem } from '@/components/v1/ui/radio-group';
import { Textarea } from '@/components/v1/ui/textarea';
import { OrganizationOnboardingSDK } from '@/lib/api/generated/control-plane/data-contracts';
import { ArrowLeftIcon, ArrowRightIcon } from '@radix-ui/react-icons';
import { useCallback, useState } from 'react';
import { IconType } from 'react-icons';
import { BiLogoGoLang, BiLogoPython, BiLogoTypescript } from 'react-icons/bi';
import { DiRuby } from 'react-icons/di';

const WHAT_TO_BUILD_MAX_LENGTH = 1000;

const SDK_OPTIONS: {
  value: OrganizationOnboardingSDK;
  label: string;
  icon: IconType;
}[] = [
  {
    value: OrganizationOnboardingSDK.PYTHON,
    label: 'Python',
    icon: BiLogoPython,
  },
  {
    value: OrganizationOnboardingSDK.TYPESCRIPT,
    label: 'TypeScript',
    icon: BiLogoTypescript,
  },
  { value: OrganizationOnboardingSDK.GO, label: 'Go', icon: BiLogoGoLang },
  { value: OrganizationOnboardingSDK.RUBY, label: 'Ruby', icon: DiRuby },
];

export type OrganizationOnboardingAnswers = {
  whatToBuild?: string;
  sdk?: OrganizationOnboardingSDK;
};

type OrganizationOnboardingQuestionsFormProps = {
  isSaving: boolean;
  defaultAnswers?: OrganizationOnboardingAnswers;
  onSubmit: (values: OrganizationOnboardingAnswers) => void;
  // Receives the current answers so the parent can restore them if the user
  // comes back to this step.
  onBack: (values: OrganizationOnboardingAnswers) => void;
};

export function OrganizationOnboardingQuestionsForm({
  isSaving,
  defaultAnswers,
  onSubmit,
  onBack,
}: OrganizationOnboardingQuestionsFormProps) {
  const [whatToBuild, setWhatToBuild] = useState(
    defaultAnswers?.whatToBuild ?? '',
  );
  const [sdk, setSdk] = useState<OrganizationOnboardingSDK | undefined>(
    defaultAnswers?.sdk,
  );

  const answers = useCallback((): OrganizationOnboardingAnswers => {
    const trimmedWhatToBuild = whatToBuild.trim();

    return {
      ...(trimmedWhatToBuild ? { whatToBuild: trimmedWhatToBuild } : {}),
      ...(sdk ? { sdk } : {}),
    };
  }, [whatToBuild, sdk]);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      onSubmit(answers());
    },
    [answers, onSubmit],
  );

  return (
    <form onSubmit={handleSubmit} className="grid gap-6 max-w-lg w-full">
      <div className="grid gap-2">
        <Label htmlFor="what-to-build">
          What would you like to build with Hatchet?
        </Label>
        <p className="text-sm text-muted-foreground">
          Optional — this helps us point you at the right docs and examples.
        </p>
        <Textarea
          id="what-to-build"
          placeholder="e.g. background task for processing video files"
          rows={4}
          maxLength={WHAT_TO_BUILD_MAX_LENGTH}
          value={whatToBuild}
          onChange={(e) => setWhatToBuild(e.target.value)}
          disabled={isSaving}
        />
      </div>

      <div className="grid gap-2">
        <Label id="sdk-question">Which SDK are you planning to use?</Label>
        <RadioGroup
          aria-labelledby="sdk-question"
          value={sdk ?? ''}
          onValueChange={(value) => setSdk(value as OrganizationOnboardingSDK)}
          disabled={isSaving}
        >
          {SDK_OPTIONS.map((option) => (
            <RadioGroupCardItem key={option.value} value={option.value}>
              <span className="flex items-center gap-2 text-sm font-medium">
                <option.icon className="size-5" />
                {option.label}
              </span>
            </RadioGroupCardItem>
          ))}
        </RadioGroup>
      </div>

      <div className="flex gap-3">
        <Button
          type="button"
          variant="outline"
          onClick={() => onBack(answers())}
          disabled={isSaving}
        >
          <ArrowLeftIcon className="mr-2 size-4" />
          Back
        </Button>
        <Button type="submit" className="flex-1" disabled={isSaving}>
          {isSaving ? (
            <>
              <Spinner />
              Getting started...
            </>
          ) : (
            <>
              Get started
              <ArrowRightIcon className="ml-2 size-4" />
            </>
          )}
        </Button>
      </div>
    </form>
  );
}
