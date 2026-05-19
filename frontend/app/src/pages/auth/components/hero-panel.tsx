import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';

// TODO-DESIGN
export function HeroPanel() {
  return (
    <div className="relative flex h-full w-full flex-col justify-between">
      <div className="relative">
        <HatchetLogo className="h-6 w-auto" />
      </div>

      {/* bottom-right hero copy (marketing-style) */}
      <div className="relative flex z-4 isolate w-full justify-end pb-10 lg:pb-12">
        <div className="max-w-2xl text-right">
          <h1 className="text-5xl font-semibold tracking-[-0.02em] dark:text-[#b8d9ff] text-[#000000] lg:text-6xl [text-wrap:balance]">
            The last orchestrator you’ll ever need
          </h1>
          <p className="mt-4 text-base text-balance dark:text-[#b8d9ff77] dark:mix-blend-plus-lighter">
            Run fast and reliable workflows for background jobs and agents.
          </p>
        </div>
      </div>
    </div>
  );
}
