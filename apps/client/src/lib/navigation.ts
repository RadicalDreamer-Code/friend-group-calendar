import { router, type Href } from 'expo-router';

/** Back to the previous screen, or to `fallback` when the screen was opened directly (e.g. a web link). */
export function goBack(fallback: Href) {
  if (router.canGoBack()) router.back();
  else router.replace(fallback);
}

/**
 * Up to a parent screen: pops back to it when it is in the stack, and opens
 * it in place of the current screen when the current one was opened directly.
 * Unlike goBack, a deep link never falls through to the screen below.
 */
export function backTo(parent: Href) {
  router.dismissTo(parent);
}
