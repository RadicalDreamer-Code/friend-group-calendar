import { useEffect, useRef } from 'react';

/** Web: calls `onEscape` when Escape is pressed, so sheets close from the keyboard like dialogs do. */
export function useEscapeKey(onEscape: () => void) {
  const latest = useRef(onEscape);
  useEffect(() => {
    latest.current = onEscape;
  });
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') latest.current();
    };
    // Capture phase: react-native-web's text inputs stop key events from bubbling up.
    window.addEventListener('keydown', onKeyDown, true);
    return () => window.removeEventListener('keydown', onKeyDown, true);
  }, []);
}
