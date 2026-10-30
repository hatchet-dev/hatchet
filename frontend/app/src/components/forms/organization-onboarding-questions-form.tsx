import { Button } from '@/components/v1/ui/button';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { Textarea } from '@/components/v1/ui/textarea';
import { OrganizationSignupAttribution } from '@/lib/api/generated/control-plane/data-contracts';
import { cn } from '@/lib/utils';
import { ArrowLeftIcon, ArrowRightIcon } from '@radix-ui/react-icons';
import { useCallback, useState } from 'react';

const ATTRIBUTION_OTHER_MAX_LENGTH = 500;

type AttributionOption = {
  value: OrganizationSignupAttribution;
  label: string;
};

// The values are stable keys the control plane forwards to the signup Slack
// thread and analytics (it holds the matching labels); only the labels here are
// display copy.
const ATTRIBUTION_OPTIONS: AttributionOption[] = [
  { value: OrganizationSignupAttribution.Search, label: 'Search engine' },
  { value: OrganizationSignupAttribution.XTwitter, label: 'X / Twitter' },
  { value: OrganizationSignupAttribution.Linkedin, label: 'LinkedIn' },
  { value: OrganizationSignupAttribution.HackerNews, label: 'Hacker News' },
  { value: OrganizationSignupAttribution.Reddit, label: 'Reddit' },
  { value: OrganizationSignupAttribution.Github, label: 'GitHub' },
  { value: OrganizationSignupAttribution.BlogArticle, label: 'A blog post' },
  {
    value: OrganizationSignupAttribution.FriendColleague,
    label: 'Friend or colleague recommendation',
  },
  {
    value: OrganizationSignupAttribution.ConferenceEvent,
    label: 'A conference or meetup',
  },
  {
    value: OrganizationSignupAttribution.AiAssistant,
    label: 'AI assistant / coding agent',
  },
  {
    value: OrganizationSignupAttribution.SearchForAlternative,
    label: 'Looking for an alternative to another tool',
  },
  { value: OrganizationSignupAttribution.Other, label: 'Other' },
];

// Randomize the display order to avoid order bias, but always pin "other"
// last. Callers should compute this once (e.g. useMemo) and hold it stable so
// the order does not reshuffle on re-render or when stepping back into the
// form.
export function shuffledAttributionOptions(): AttributionOption[] {
  const rest = ATTRIBUTION_OPTIONS.filter(
    (o) => o.value !== OrganizationSignupAttribution.Other,
  );
  for (let i = rest.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [rest[i], rest[j]] = [rest[j], rest[i]];
  }
  const other = ATTRIBUTION_OPTIONS.find(
    (o) => o.value === OrganizationSignupAttribution.Other,
  );
  return other ? [...rest, other] : rest;
}

export type OrganizationOnboardingAnswers = {
  attribution?: OrganizationSignupAttribution[];
  attributionOther?: string;
};

type OrganizationOnboardingQuestionsFormProps = {
  isSaving: boolean;
  // A stable, pre-shuffled option order (see shuffledAttributionOptions).
  options: AttributionOption[];
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
  const [attribution, setAttribution] = useState<
    OrganizationSignupAttribution[]
  >(defaultAnswers?.attribution ?? []);
  const [attributionOther, setAttributionOther] = useState(
    defaultAnswers?.attributionOther ?? '',
  );

  const otherSelected = attribution.includes(
    OrganizationSignupAttribution.Other,
  );

  const toggle = (value: OrganizationSignupAttribution) =>
    setAttribution((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value],
    );

  const answers = useCallback((): OrganizationOnboardingAnswers => {
    if (attribution.length === 0) {
      return {};
    }

    const trimmedOther = attributionOther.trim();

    return {
      attribution,
      ...(otherSelected && trimmedOther
        ? { attributionOther: trimmedOther }
        : {}),
    };
  }, [attribution, attributionOther, otherSelected]);

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
          We don't use tracking pixels. Instead, we humbly ask that you share a
          bit more about where you heard about us. Thanks in advance!
        </p>
        {/* Multi-select chips that flow left to right and wrap. Toggle
            buttons with aria-pressed rather than a radio group, since more
            than one source can apply. */}
        <div
          role="group"
          aria-labelledby="attribution-question"
          className="flex flex-wrap gap-2"
        >
          {options.map((option) => {
            const selected = attribution.includes(option.value);
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={selected}
                onClick={() => toggle(option.value)}
                disabled={isSaving}
                className={cn(
                  'rounded-md border px-2.5 py-1.5 text-left text-xs ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
                  // The hover border only applies to unselected chips. On a
                  // selected chip it would override the selection outline, so
                  // the highlight would not appear until the mouse left.
                  selected
                    ? 'border-primary bg-muted/40'
                    : 'border-border/50 bg-muted/20 hover:border-border',
                )}
              >
                {option.label}
              </button>
            );
          })}
        </div>
        {otherSelected && (
          <Textarea
            id="attribution-other"
            aria-label="Tell us more about where you heard about us"
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
          <Button
            type="submit"
            size="sm"
            disabled={isSaving || attribution.length === 0}
          >
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
