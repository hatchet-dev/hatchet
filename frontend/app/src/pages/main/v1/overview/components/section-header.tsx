import { OnboardingBadge } from './onboarding-badge';
import { Separator } from '@/components/v1/ui/separator';

export function SectionHeader({
  title,
  showOnboardingBadge = false,
  completed = false,
}: {
  title: string;
  showOnboardingBadge?: boolean;
  completed?: boolean;
}) {
  return (
    <>
      <span className="inline-flex items-baseline gap-5">
        <h2 className="text-md">{title}</h2>
        {showOnboardingBadge && <OnboardingBadge completed={completed} />}
      </span>
      <Separator className="my-4 bg-border/50" flush />
    </>
  );
}
