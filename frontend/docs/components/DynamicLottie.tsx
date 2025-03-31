import dynamic from 'next/dynamic';
import { ComponentProps } from 'react';

// Dynamically import the Lottie component with SSR disabled
const LottiePlayer = dynamic(
  () => import('react-lottie-player').then((mod) => mod.default),
  { ssr: false }
);

export type DynamicLottieProps = ComponentProps<typeof LottiePlayer>;

// Export the dynamic component with the same props as the original
function DynamicLottie(props: DynamicLottieProps) {
  return <LottiePlayer {...props} />;
}

// Using CommonJS export
export default DynamicLottie; 