import { Button } from '@/components/v1/ui/button';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { RadioGroup, RadioGroupCardItem } from '@/components/v1/ui/radio-group';
import { Textarea } from '@/components/v1/ui/textarea';
import { CreateOrganizationRequest } from '@/lib/api/generated/control-plane/data-contracts';
import { ArrowLeftIcon, ArrowRightIcon } from '@radix-ui/react-icons';
import { useCallback, useState } from 'react';

const ATTRIBUTION_OTHER_MAX_LENGTH = 500;

// The stable value keys are what the control plane stores and the Slack message
// references; only the labels are display copy. Keep in sync with the backend
// (docs/plans/signup-attribution-copy.mdx).
type AttributionValue = NonNullable<CreateOrganizationRequest['attribution']>;

const ATTRIBUTION_OPTIONS: { value: AttributionValue; label: string }[] = [
  { value: 'search', label: 'Search engine' },
  { value: 'x_twitter', label: 'X / Twitter' },
  { value: 'linkedin', label: 'LinkedIn' },
  { value: 'hacker_news', label: 'Hacker News' },
  { value: 'reddit', label: 'Reddit' },
  { value: 'github', label: 'GitHub' },
  { value: 'blog_article', label: 'A blog post' },
  { value: 'friend_colleague', label: 'Friend or colleague recommendation' },
  { value: 'conference_event', label: 'A conference or meetup' },
  { value: 'ai_assistant', label: 'An AI assistant recommended it' },
  {
    value: 'search_for_alternative',
    label: 'Looking for an alternative to another tool',
  },
  { value: 'other', label: 'Other' },
];

// Randomize the display order per render to avoid order bias, but always pin
// "other" last. Callers should compute this once (e.g. useMemo) and hold it
// stable so the order does not reshuffle on re-render or when stepping back
// into the form.
export function shuffledAttributionOptions(): {
  value: AttributionValue;
  label: string;
}[] {
  const rest = ATTRIBUTION_OPTIONS.filter((o) => o.value !== 'other');
  for (let i = rest.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [rest[i], rest[j]] = [rest[j], rest[i]];
  }
  const other = ATTRIBUTION_OPTIONS.find((o) => o.value === 'other');
  return other ? [...rest, other] : rest;
}

export type OrganizationOnboardingAnswers = {
  attribution?: AttributionValue;
  attributionOther?: string;
};

type OrganizationOnboardingQuestionsFormProps = {
  isSaving: boolean;
  // A stable, pre-shuffled option order (see shuffledAttributionOptions).
  options: { value: AttributionValue; label: string }[];
  defaultAnswers?: OrganizationOnboardingAnswers;
  onSubmit: (values: OrganizationOnboardingAnswers) => void;
  // Receives the current answers so the parent can restore them if the user
  // comes back to this step.
  onBack: (values: OrganizationOnboardingAnswers) => void;
};

export function OrganizationOnboardingQuestionsForm({
  isSaving,
  options,
  defaultAnswers,
  onSubmit,
  onBack,
}: OrganizationOnboardingQuestionsFormProps) {
  const [attribution, setAttribution] = useState<AttributionValue | undefined>(
    defaultAnswers?.attribution,
  );
  const [attributionOther, setAttributionOther] = useState(
    defaultAnswers?.attributionOther ?? '',
  );

  const answers = useCallback((): OrganizationOnboardingAnswers => {
    if (!attribution) {
      return {};
    }

    const trimmedOther = attributionOther.trim();

    return {
      attribution,
      ...(attribution === 'other' && trimmedOther
        ? { attributionOther: trimmedOther }
        : {}),
    };
  }, [attribution, attributionOther]);

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
        <Label id="attribution-question">How did you hear about us?</Label>
        <p className="text-sm text-muted-foreground">
          This helps us understand how people find Hatchet. Pick the closest.
        </p>
        <RadioGroup
          aria-labelledby="attribution-question"
          value={attribution ?? ''}
          onValueChange={(value) => setAttribution(value as AttributionValue)}
          disabled={isSaving}
        >
          {options.map((option) => (
            <RadioGroupCardItem key={option.value} value={option.value}>
              <span className="text-sm font-medium">{option.label}</span>
            </RadioGroupCardItem>
          ))}
        </RadioGroup>
        {attribution === 'other' && (
          <Textarea
            id="attribution-other"
            placeholder="Tell us more (optional)"
            rows={2}
            maxLength={ATTRIBUTION_OTHER_MAX_LENGTH}
            value={attributionOther}
            onChange={(e) => setAttributionOther(e.target.value)}
            disabled={isSaving}
          />
        )}
      </div>

      <div className="flex items-center gap-3">
        <Button
          type="button"
          variant="outline"
          onClick={() => onBack(answers())}
          disabled={isSaving}
        >
          <ArrowLeftIcon className="mr-2 size-4" />
          Back
        </Button>
        <div className="ml-auto flex items-center gap-3">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="text-muted-foreground"
            onClick={() => onSubmit({})}
            disabled={isSaving}
          >
            Prefer not to say
          </Button>
          <Button type="submit" size="sm" disabled={isSaving || !attribution}>
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
      </div>
    </form>
  );
}
