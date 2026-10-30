import { useHotkey } from '@tanstack/react-hotkeys';
import type React from 'react';

/**
 * Opens the switcher on ⌘K / Ctrl+K. The top nav mounts one switcher per
 * breakpoint, so both register the hotkey and only the one whose trigger is
 * visible and enabled reacts.
 */
export function useTenantSwitcherShortcut(
  triggerRef: React.RefObject<HTMLButtonElement>,
  open: () => void,
) {
  useHotkey(
    'Mod+K',
    () => {
      const trigger = triggerRef.current;
      if (!trigger || trigger.disabled || trigger.offsetParent === null) {
        return;
      }
      open();
    },
    { conflictBehavior: 'allow' },
  );
}
