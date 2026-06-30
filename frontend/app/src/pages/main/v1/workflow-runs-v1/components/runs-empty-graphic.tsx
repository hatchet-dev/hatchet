import b1404 from '@/assets/illustrations/b1-404.svg';
import b2404 from '@/assets/illustrations/b2-404.svg';
import { useEffect, useState } from 'react';

export function RunsEmptyGraphic({ className = 'h-24 w-auto' }: { className?: string }) {
  const [frame, setFrame] = useState(0);
  useEffect(() => {
    const id = setInterval(() => setFrame((f) => (f + 1) % 2), 300);
    return () => clearInterval(id);
  }, []);
  return (
    <img
      src={frame === 0 ? b1404 : b2404}
      alt=""
      aria-hidden="true"
      className={className}
    />
  );
}
