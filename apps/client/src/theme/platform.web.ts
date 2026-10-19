import { flushSync } from 'react-dom';

import { motion, type ColorScheme, type Colors } from './tokens';

type DocumentWithTransitions = Document & {
  startViewTransition?: (update: () => void) => unknown;
};

const prefersReducedMotion = () =>
  typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;

/**
 * Web: crossfade the whole page between the old and new theme with the
 * View Transitions API. Browsers without it, and users who prefer reduced
 * motion, get an instant switch.
 */
export function withViewTransition(update: () => void) {
  const doc = document as DocumentWithTransitions;
  if (!doc.startViewTransition || prefersReducedMotion()) {
    update();
    return;
  }
  doc.startViewTransition(() => flushSync(update));
}

let baseStylesInjected = false;

function injectBaseStyles() {
  if (baseStylesInjected) return;
  baseStylesInjected = true;
  const style = document.createElement('style');
  style.setAttribute('data-app', 'base');
  style.textContent = `
    html, body { -webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale; }
    body { -webkit-tap-highlight-color: transparent; overscroll-behavior-y: none; }
    ::view-transition-old(root), ::view-transition-new(root) {
      animation-duration: ${motion.duration.slow}ms;
      animation-timing-function: cubic-bezier(0.2, 0, 0, 1);
    }
  `;
  document.head.appendChild(style);
}

/** Web: keep the page behind the app, scrollbars and browser chrome in sync with the theme. */
export function applyDocumentTheme(scheme: ColorScheme, colors: Colors) {
  injectBaseStyles();
  const root = document.documentElement;
  root.style.colorScheme = scheme;
  root.style.backgroundColor = colors.background;
  document.body.style.backgroundColor = colors.background;

  let meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]');
  if (!meta) {
    meta = document.createElement('meta');
    meta.name = 'theme-color';
    document.head.appendChild(meta);
  }
  meta.content = colors.background;
}
