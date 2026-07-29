import hatchetSpinners from '@/assets/illustrations/hatchet-5.svg';

export function RunsEmptyGraphic({
  className = 'h-24 w-auto',
}: {
  className?: string;
}) {
  return (
    <img
      src={hatchetSpinners}
      alt=""
      aria-hidden="true"
      className={className}
    />
  );
}
