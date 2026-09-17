import { type UseCaseChoice } from './onboarding-steps-types';

// Hatchet's own marketing "totem" illustrations, copied verbatim from the
// marketing site's SVG set. Each is a wide horizontal strip (336x48). The
// currentColor strokes inherit the card's text color so selection can
// dim/brighten the linework; the brand-hex strokes stay fixed.

type TotemProps = { className?: string };

export function TotemAgents({ className }: TotemProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 336 48"
      className={className}
      aria-hidden
    >
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M0 24h48M56 24h5m-13 0h1m16 0h5m4 0h5m4 0h5m7 0h1m-47 2v-4h4v4zm42 0v-4h4v4z"
      />
      <path
        stroke="#3392ff"
        strokeWidth="1.5"
        d="M104 30h32m-32-4h32m-32-4h32m-32-4h32"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M104.5 34h-4V14h4m-4 10H96m39.5-10h4v10m-4 10h4V24m0 0h4.5M168 36V12m-12 12h24m4 0c0-8.837-7.163-16-16-16s-16 7.163-16 16m32 0c0 8.837-7.163 16-16 16s-16-7.163-16-16m32 0h8m-40 0h-8m4 16h3m1-4v3m1 1h3m-4 1v3m6-4a6 6 0 1 1-12 0 6 6 0 0 1 12 0Zm32-32a6 6 0 1 1-12 0 6 6 0 0 1 12 0Zm-3 0a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
      />
      <path
        stroke="#3392ff"
        strokeWidth="1.5"
        d="M204 14v20m4-20v20m4-20v20m4-20v4m0 2v8m0 2v4m4-20v4m4-4v4m4-4v4m-8 3v6m4-5v4m4-4v4m-8 4v4m4-4v4m4-4v4"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M200.5 34h-4V14h4m-4 10H192m39.5-10h4v10m-4 10h4V24m0 0h4.5M240 24h24m-4-4 4 4-4 4m5.5 0 4-4m0 0-4-4m4 4H288M288 24h48"
      />
    </svg>
  );
}

export function TotemDurable({ className }: TotemProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 336 48"
      className={className}
      aria-hidden
    >
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M0 24h48M48 24h24m-4-4 4 4-4 4m5.5 0 4-4m0 0-4-4m4 4H96"
      />
      <path
        stroke="#bc46dd"
        strokeWidth="1.5"
        d="M96 24c.347 0 .52 0 .648-.011.735-.063 1.095-.21 1.662-.682.099-.082.31-.292.735-.71q3.832-3.786 7.663 1.403 4.43 6 8.861 0t8.862 0q4.43 6 8.861 0 3.832-5.188 7.663-1.404c.424.42.636.628.735.71.567.472.927.62 1.662.683.128.011.301.011.648.011"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M96 24c.347 0 .52 0 .648.011.735.063 1.095.21 1.662.682.099.082.31.292.735.71q3.832 3.786 7.663-1.403 4.43-6 8.861 0t8.862 0q4.43-6 8.861 0 3.832 5.189 7.663 1.404c.424-.42.636-.628.735-.71.567-.472.927-.62 1.662-.683.128-.011.301-.011.648-.011M152 16h8m-8 0V8h32v8m-32 0v8m8-8h8m-8 0v-6m8 6h8m-8 0v-6m8 6h8m-8 0v-6m8 6v8m-40 0h8m0 0h32m-32 0v8m32-8h8m-8 0v8m-32 0h8m-8 0v8h32v-8m-24 0h8m-8 0v6m8-6h8m-8 0v6m8-6h8m-8 0v6m-16-8v-4m8 4v-4m8 4v-4m-16-4v-4m8 4v-4m8 4v-4"
      />
      <path
        stroke="#bc46dd"
        strokeWidth="1.5"
        d="M180 34.75a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Zm-24-8a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Zm16 0a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Zm-16-8a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Zm8 0a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Zm-8-8a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Zm8 0a1.25 1.25 0 1 1 0 2.5 1.25 1.25 0 0 1 0-2.5Z"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M208 24h-16m16 0 4 4h8l4-4m-16 0 4-4h8l4 4m16 0h-16M240 24h24m-4-4 4 4-4 4m5.5 0 4-4m0 0-4-4m4 4H288M288 24h48"
      />
    </svg>
  );
}

export function TotemParallel({ className }: TotemProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 336 48"
      className={className}
      aria-hidden
    >
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M0 24h48M48 24h24m-4-4 4 4-4 4m5.5 0 4-4m0 0-4-4m4 4H96M96 24h19m-9 4h9m-9-8h9m10 4h19m-19-4h9m-9 8h9m-14 8V12"
      />
      <path
        stroke="#bad61c"
        strokeWidth="1.5"
        d="M156 32v8.5m4-17v17m4-26.5v26.5m4-22.5v22.5m4-22.5v22.5m4-14.5v14.5m4-18.5v18.5"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M144 24h8m0 0V12a4 4 0 0 1 4-4h24a4 4 0 0 1 4 4v12m-32 0v16h32V24m0 0h8m-32-14v4m8-4v4m8-4v4m-16 16v-4m0-4v-4m8 4v-4m8 4v-4m-8 12v-4m8 4v-4m-16 12v-4m8 4v-4m8 4v-4"
      />
      <path
        stroke="#bad61c"
        strokeWidth="1.5"
        d="M192 24c.347 0 .52 0 .648-.011.735-.063 1.095-.21 1.662-.682.099-.082.311-.292.735-.71q3.832-3.786 7.663 1.403 4.43 6 8.861 0t8.862 0q4.43 6 8.861 0 3.831-5.188 7.663-1.404c.424.42.636.628.735.71.567.472.927.62 1.662.683.128.011.301.011.648.011"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M192 24c.347 0 .52 0 .648.011.735.063 1.095.21 1.662.682.099.082.311.292.735.71q3.832 3.786 7.663-1.403 4.43-6 8.861 0t8.862 0q4.43-6 8.861 0 3.831 5.189 7.663 1.404c.424-.42.636-.628.735-.71.567-.472.927-.62 1.662-.683.128-.011.301-.011.648-.011M240 24h48M288 24h48"
      />
    </svg>
  );
}

export function TotemManagement({ className }: TotemProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 336 48"
      className={className}
      aria-hidden
    >
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M0 24h48M48 24h24m-4-4 4 4-4 4m5.5 0 4-4m0 0-4-4m4 4H96M120 36V12m-12 12h24m4 0c0-8.837-7.163-16-16-16s-16 7.163-16 16m32 0c0 8.837-7.163 16-16 16s-16-7.163-16-16m32 0h8m-40 0h-8M145 24c0-7.723 5.472-14.168 12.75-15.67A16 16 0 0 1 161 8h14c8.837 0 16 7.164 16 16m0 0c0 8.837-7.163 16-16 16h-14c-1.114 0-2.201-.114-3.25-.33M191 24h1m-47 0h-1m1 0c0 7.723 5.472 14.168 12.75 15.67M171 14h-4v20h4m6-20h4v20h-4m-19.25-18V8.33m0 31.34V32M161 21v6h-6v-6z"
      />
      <path
        stroke="#006e8c"
        strokeWidth="1.5"
        d="M170.5 18h7m-7 4h7m-7 4h7m-7 4h7"
      />
      <path
        stroke="currentColor"
        strokeWidth="1.5"
        d="M199 8h34m-34 4h34m-34 4h34m-34 4h34m-41 4h48m-41 4h34m-34 4h34m-34 4h34m-34 4h34M240 24h24m-4-4 4 4-4 4m5.5 0 4-4m0 0-4-4m4 4H288M288 24h48"
      />
    </svg>
  );
}

// Map each onboarding use case to a totem. There are only four totems for six
// use cases, so scheduled/durable share TotemDurable and simple/custom share
// TotemAgents. That sharing is expected.
export const useCaseTotems: Record<
  UseCaseChoice,
  (props: TotemProps) => JSX.Element
> = {
  simple: TotemAgents,
  scheduled: TotemDurable,
  fanout: TotemParallel,
  event: TotemManagement,
  durable: TotemDurable,
  custom: TotemAgents,
};
