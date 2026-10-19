import * as SystemUI from 'expo-system-ui';

import type { ColorScheme, Colors } from './tokens';

/** Native: theme switches apply instantly (a crossfade is planned for the motion polish milestone). */
export function withViewTransition(update: () => void) {
  update();
}

/** Native: paint the root view so no white shows behind screens during transitions. */
export function applyDocumentTheme(_scheme: ColorScheme, colors: Colors) {
  SystemUI.setBackgroundColorAsync(colors.background).catch(() => {});
}
