import { useEffect, useState } from 'react';
import { cn } from '@/next/lib/utils';

interface PulseIndicatorProps {
  value: any;
  className?: string;
  label?: string;
}

const COLORS = [
  'bg-blue-500',
  'bg-green-500',
  'bg-purple-500',
  'bg-pink-500',
  'bg-orange-500',
  'bg-cyan-500',
];

const PULSE_COLORS = [
  'bg-blue-500/20',
  'bg-green-500/20',
  'bg-purple-500/20',
  'bg-pink-500/20',
  'bg-orange-500/20',
  'bg-cyan-500/20',
];

export function PulseIndicator({ value, className, label }: PulseIndicatorProps) {
  const [pulseKey, setPulseKey] = useState(0);
  const [dotColor, setDotColor] = useState(COLORS[0]);
  const [pulseColor, setPulseColor] = useState(PULSE_COLORS[0]);

  useEffect(() => {
    setPulseKey(Date.now());
    setDotColor(COLORS[Math.floor(Math.random() * COLORS.length)]);
    setPulseColor(PULSE_COLORS[Math.floor(Math.random() * PULSE_COLORS.length)]);
  }, [value]);

  return (
    <span className={cn("flex items-center gap-2", className)}>
      {label && <span className="text-xs text-muted-foreground">{label}</span>}
      <div key={pulseKey} className="relative flex h-2 w-2">
        <span className={cn("absolute inline-flex h-full w-full rounded-full scale-0 blur-sm animate-[ping_1s_ease-in-out]", pulseColor)}></span>
        <span className={cn("relative inline-flex h-2 w-2 rounded-full", dotColor)}></span>
      </div>
    </span>
  );
} 