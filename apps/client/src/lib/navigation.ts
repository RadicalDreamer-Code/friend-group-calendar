import { router, type Href } from 'expo-router';

/** Back to the previous screen, or to `fallback` when the screen was opened directly (e.g. a web link). */
export function goBack(fallback: Href) {
  if (router.canGoBack()) router.back();
  else router.replace(fallback);
}
