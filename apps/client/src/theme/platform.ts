import * as SystemUI from 'expo-system-ui';

import type { ColorScheme, Colors } from './tokens';

/** Native: theme switches apply instantly; there is no view transition to crossfade with. */
export function withViewTransition(update: () => void) {
  update();
}

/** Native: paint the root view so no white shows behind screens during transitions. */
export function applyDocumentTheme(_scheme: ColorScheme, colors: Colors) {
  SystemUI.setBackgroundColorAsync(colors.background).catch(() => {});
}
