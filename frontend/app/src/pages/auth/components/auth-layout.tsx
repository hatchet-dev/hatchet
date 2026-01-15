import { HeroPanel } from './hero-panel';
import { CSSProperties, PropsWithChildren } from 'react';

export function AuthLayout({ children }: PropsWithChildren) {
  const bgContentStyle = {
    '--noise-url': 'url("/noise.png")',
  } as CSSProperties;

  return (
    <div className="min-h-screen w-full lg:grid lg:grid-cols-[1fr_round(nearest,50%,24px)]">
      <div className="relative hidden overflow-hidden bg-muted/30 px-10 py-12 lg:flex">
        <div
          className="
          [--color1:color-mix(in_srgb,_hsl(287_69%_57%)_calc(100%_*_1),_transparent)]
          [--color2:color-mix(in_srgb,_hsl(212_100%_60%)_calc(100%_*_.56),_transparent)]
          [--direction:bottom] [--scale:5] [--gridSize:24px] [--n:6] [--t:1px] [--g:2px] [--c:#fff7_25%,#0000_0]

          pointer-events-none absolute left-0 bottom-0 w-full overflow-clip
          [height:clamp(50rlh,30vh,50rlh)]
          [background-color:#b8d9ff14]
          [mask-image:linear-gradient(to_var(--direction),transparent_50%,white_100%),var(--noise-url),conic-gradient(at_var(--g)_var(--t),var(--c)),conic-gradient(from_180deg_at_var(--t)_var(--g),var(--c))]
          [mask-position:0_0,0%_0%,calc((var(--gridSize)_/_var(--n)_-_var(--g)_+_var(--t))_/_2)_0,round(up,100%,calc(var(--g)*2))_calc((var(--gridSize)_/_var(--n)_-_var(--g)_+_var(--t))_/_2)]
          [mask-size:100%,calc(300%*var(--scale))_calc(300%*var(--scale)),calc(var(--gridSize)_/_var(--n))_var(--gridSize),var(--gridSize)_calc(var(--gridSize)_/_var(--n))]
          [mask-composite:intersect,add,add,subtract]

          before:absolute before:left-0 before:top-0 before:h-full before:w-full before:opacity-0 before:content-[''] before:bg-[repeating-radial-gradient(circle_at_50%_50%,transparent_0%,var(--color1)_20%,var(--color2)_25%,transparent_32%)] before:animate-[heroPulse_7s_linear_infinite] before:[animation-delay:3.5s]

          after:absolute after:left-0 after:top-0 after:h-full after:w-full after:opacity-0 after:content-[''] after:bg-[repeating-radial-gradient(circle_at_50%_50%,transparent_0%,var(--color2)_9%,var(--color1)_14%,transparent_21%)] after:animate-[heroPulse_6s_linear_infinite]"
          style={bgContentStyle}
        />
        <HeroPanel />
      </div>

      <div className="w-full overflow-y-auto">
        <div className="flex min-h-screen w-full items-center justify-center px-4 py-10 lg:justify-start lg:px-12 lg:py-12">
          <div className="w-full max-w-lg">
            <div className="flex w-full flex-col justify-center space-y-6">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
