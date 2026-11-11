import styles from './banana-mode.module.css';
import { useBananaMode, useBananaRain } from './use-banana-mode';
import { useEffect, useState, type CSSProperties } from 'react';
import { createPortal } from 'react-dom';

type Raindrop = {
  id: string;
  left: string;
  delay: string;
  duration: string;
  size: string;
};

function raindrops(
  count: number,
  seed: number,
  durationBase: number,
): Raindrop[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `${seed}-${i}`,
    left: `${(i * seed) % 100}%`,
    delay: `${((i * 13) % 40) / 10}s`,
    duration: `${durationBase + ((i * 7) % 22) / 10}s`,
    size: `${1.1 + ((i * 11) % 16) / 10}rem`,
  }));
}

const AMBIENT = raindrops(16, 37, 7);
const BURST = raindrops(40, 19, 2.4);
const BURST_MS = 7000;

function Rain({ drops }: { drops: Raindrop[] }) {
  return drops.map((drop) => (
    <span
      key={drop.id}
      className={styles.banana}
      style={
        {
          '--left': drop.left,
          '--delay': drop.delay,
          '--duration': drop.duration,
          '--size': drop.size,
        } as CSSProperties
      }
    >
      🍌
    </span>
  ));
}

export function BananaMode() {
  const [enabled] = useBananaMode();
  const [rain] = useBananaRain();
  const raining = enabled && rain;
  const [burst, setBurst] = useState(false);

  useEffect(() => {
    if (enabled) {
      // One attribute on <html> lets the stylesheet restyle the shell without
      // editing shared theme tokens. Removing it restores the previous theme.
      document.documentElement.dataset.bananaMode = 'on';
    } else {
      delete document.documentElement.dataset.bananaMode;
    }

    return () => {
      delete document.documentElement.dataset.bananaMode;
    };
  }, [enabled]);

  useEffect(() => {
    if (!raining) {
      return () => {};
    }

    let buffer = '';
    let timer: ReturnType<typeof setTimeout> | undefined;
    const onKey = (event: KeyboardEvent) => {
      if (event.key.length !== 1) {
        return;
      }

      buffer = (buffer + event.key.toLowerCase()).slice(-6);
      if (buffer !== 'banana') {
        return;
      }

      buffer = '';
      setBurst(true);
      if (timer) {
        clearTimeout(timer);
      }
      timer = setTimeout(() => setBurst(false), BURST_MS);
    };

    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('keydown', onKey);
      if (timer) {
        clearTimeout(timer);
      }
      setBurst(false);
    };
  }, [raining]);

  if (!raining) {
    return null;
  }

  return createPortal(
    <div className={styles.overlay} aria-hidden>
      <Rain drops={AMBIENT} />
      {burst ? <Rain drops={BURST} /> : null}
    </div>,
    document.body,
  );
}
