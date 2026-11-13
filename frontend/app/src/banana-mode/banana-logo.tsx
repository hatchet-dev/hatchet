import { cn } from '@/lib/utils';
import React from 'react';

type BananaLogoProps = React.SVGAttributes<SVGSVGElement> & {
  title?: string;
  variant?: 'lockup' | 'mark' | 'wordmark';
};

const WORDMARK_GLYPHS = {
  A: {
    start: 55.4688,
    width: 12.995,
    d: 'M68.4639 20.6255H64.3926L63.9326 18.3022H59.7461L59.2861 20.6255H55.4688L59.7461 3.37549H64.1855L68.4639 20.6255ZM60.2988 15.5425H63.4033L61.8623 7.72217L60.2988 15.5425Z',
  },
  Á: {
    start: 55.4688,
    width: 12.995,
    d: 'M68.4639 20.6255H64.3926L63.9326 18.3022H59.7461L59.2861 20.6255H55.4688L59.7461 3.37549H64.1855L68.4639 20.6255ZM60.2988 15.5425H63.4033L61.8623 7.72217L60.2988 15.5425ZM61.6 2.7L64.6 0H67L64 2.7H61.6Z',
  },
  T: {
    start: 68.9238,
    width: 11.5,
    d: 'M80.4238 6.61865H76.3994V20.6489H72.9492V6.61865H68.9238V3.39893H80.4238V6.61865Z',
  },
  C: {
    start: 82.6338,
    width: 11.9141,
    d: 'M88.5908 3.14502C92.0637 3.14506 94.5478 5.30746 94.5479 8.96436V10.1147H91.0977V9.05615C91.0975 7.42337 90.0857 6.36577 88.5908 6.36572C87.0959 6.36572 86.0841 7.42333 86.084 9.05615V14.9448C86.0841 16.5776 87.0959 17.6353 88.5908 17.6353C90.0856 17.6352 91.0975 16.5776 91.0977 14.9448V13.6108H94.5479V14.9448C94.5477 18.6476 92.0866 20.8549 88.5908 20.855C85.0949 20.855 82.634 18.6476 82.6338 14.9448V8.96436C82.6338 5.30742 85.1179 3.14502 88.5908 3.14502Z',
  },
  H: {
    start: 41,
    width: 11.8916,
    d: 'M44.4502 10.3677H49.4414V3.37549H52.8916V20.6255H49.4414V13.5874H44.4502V20.6255H41V3.37549H44.4502V10.3677Z',
  },
  E: {
    start: 112.911,
    width: 9.937,
    d: 'M122.801 6.59521H116.361V10.1831H121.812V13.4038H116.361V17.4058H122.848V20.6255H112.911V3.37549H122.801V6.59521Z',
  },
  B: {
    start: 0,
    width: 12.3,
    d: 'M0 3.37549H6.9C10.1 3.37549 11.7 4.9 11.7 7.3C11.7 8.85 11.05 9.95 9.8 10.6C11.45 11.25 12.3 12.6 12.3 14.8C12.3 18.3 10.2 20.6255 6.9 20.6255H0V3.37549ZM3.45 6.45H6.5C7.65 6.45 8.3 7 8.3 8C8.3 9 7.65 9.55 6.5 9.55H3.45V6.45ZM3.45 12.45H6.7C8 12.45 8.75 13.25 8.75 14.6C8.75 16.1 8 17.55 6.7 17.55H3.45V12.45Z',
  },
  N: {
    start: 0,
    width: 11.89,
    d: 'M0 20.6255V3.37549H3.45L8.44 13.62V3.37549H11.89V20.6255H8.44L3.45 10.38V20.6255H0Z',
  },
} as const;

const BANANA_ICON_D =
  'M17 2L19 0.5C19.9 0.2 20.5 1 20.1 1.9C26 3.2 30.5 7.8 31 13C31.3 16.5 30.3 19 28.8 21L28.4 22.6C28.1 23.4 26.9 23.3 27 22.4L27.1 21.8C26.5 17 25 11.5 22.8 8C21.3 5.6 19.4 3.7 17 2ZM21.8 4.6C26 6.2 28.6 9.7 29 13.4C29.2 15.6 28.8 17.6 28 19.3L27.4 19.1C28.1 17.5 28.4 15.6 28.2 13.5C27.8 10 25.6 6.9 21.5 5.2Z';

const BANANA_WORDMARK = 'BANÁCHET';
const LETTER_GAP = 2.1;
const WORDMARK_X = 41;

const bananaLetters = (() => {
  let x = WORDMARK_X;
  return Array.from(BANANA_WORDMARK).map((letter, i) => {
    const glyph = WORDMARK_GLYPHS[letter as keyof typeof WORDMARK_GLYPHS];
    const placed = { key: `${letter}${i}`, d: glyph.d, dx: x - glyph.start };
    x += glyph.width + LETTER_GAP;
    return placed;
  });
})();

const lastGlyph =
  WORDMARK_GLYPHS[
    Array.from(BANANA_WORDMARK).pop() as keyof typeof WORDMARK_GLYPHS
  ];
const lastLetter = bananaLetters[bananaLetters.length - 1];
const VIEWBOX_WIDTH = lastLetter.dx + lastGlyph.start + lastGlyph.width;

export function BananaLogo({
  title = 'Banáchet',
  variant = 'lockup',
  className,
  ...props
}: BananaLogoProps) {
  const isMarkOnly = variant === 'mark';
  const isWordmarkOnly = variant === 'wordmark';

  return (
    <svg
      viewBox={
        isMarkOnly
          ? '0 0 33 24'
          : isWordmarkOnly
            ? `${WORDMARK_X} 0 ${VIEWBOX_WIDTH - WORDMARK_X} 24`
            : `0 0 ${VIEWBOX_WIDTH} 24`
      }
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      role="img"
      aria-label={title}
      className={cn('text-[hsl(48,100%,62%)]', className)}
      {...props}
    >
      {!isWordmarkOnly ? (
        <>
          <path
            d={BANANA_ICON_D}
            fill="currentColor"
            fillRule="evenodd"
            transform="translate(-1.5 0)"
          />
          <path
            d={BANANA_ICON_D}
            fill="currentColor"
            fillRule="evenodd"
            transform="rotate(180 16.5 12) translate(-1.5 0)"
          />
        </>
      ) : null}
      {!isMarkOnly
        ? bananaLetters.map((letter) => (
            <path
              key={letter.key}
              d={letter.d}
              transform={`translate(${letter.dx} 0)`}
              fill="currentColor"
              fillRule="evenodd"
            />
          ))
        : null}
    </svg>
  );
}
