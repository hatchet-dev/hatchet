import { RiCheckboxCircleFill } from 'react-icons/ri';

export type OnboardingBadgeProps = {
  completed: boolean;
};

const onboardingSVG = (
  <svg
    width="12"
    height="12"
    viewBox="0 0 12 12"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    className="bottom-[1px] relative"
  >
    <path
      d="M10.499 1.5V2.5C10.499 7.3137 7.81275 9.5 4.49902 9.5L3.5482 9.49995C3.65395 7.9938 4.1227 7.0824 5.34695 5.99945C5.94895 5.4669 5.89825 5.15945 5.60145 5.33605C3.55965 6.5508 2.54557 8.1931 2.50059 10.8151L2.49902 11H1.49902C1.49902 10.3187 1.55688 9.69985 1.6719 9.1341C1.55665 8.48705 1.49902 7.6088 1.49902 6.5C1.49902 3.73857 3.7376 1.5 6.499 1.5C7.499 1.5 8.499 2 10.499 1.5Z"
      fill="hsl(var(--brand))"
    />
  </svg>
);

export function OnboardingBadge({ completed }: OnboardingBadgeProps) {
  return completed ? (
    <span className="text-[10px] font-mono tracking-widest uppercase inline-flex items-center gap-1.5 text-foreground/50">
      <RiCheckboxCircleFill />
      Onboarding step
    </span>
  ) : (
    <span className="text-[10px] font-mono tracking-widest uppercase inline-flex items-center gap-1.5 text-brand">
      {onboardingSVG}
      Onboarding step
    </span>
  );
}
