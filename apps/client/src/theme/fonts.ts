import {
  Manrope_400Regular,
  Manrope_500Medium,
  Manrope_600SemiBold,
  Manrope_700Bold,
  Manrope_800ExtraBold,
} from '@expo-google-fonts/manrope';
import type { FontFamilyDefinition } from 'expo-font';

/**
 * The two app fonts, one family each so fontWeight picks the face
 * (see fontFamilies in tokens.ts). Loaded by the root layout before the
 * splash screen hides.
 *
 * Fraunces is a variable font; these are static instances with its "soft"
 * and "wonky" axes turned all the way up (SOFT 100, WONK 1), downloaded
 * from the Google Fonts API at opsz 24 (600) and 36 (700). License:
 * assets/fonts/OFL.txt.
 */
export const appFonts: FontFamilyDefinition[] = [
  {
    fontFamily: 'Manrope',
    fontDefinitions: [
      { path: Manrope_400Regular, weight: 400 },
      { path: Manrope_500Medium, weight: 500 },
      { path: Manrope_600SemiBold, weight: 600 },
      { path: Manrope_700Bold, weight: 700 },
      { path: Manrope_800ExtraBold, weight: 800 },
    ],
  },
  {
    fontFamily: 'Fraunces',
    fontDefinitions: [
      { path: require('../../assets/fonts/Fraunces-SemiBold-Soft.ttf'), weight: 600 },
      { path: require('../../assets/fonts/Fraunces-Bold-Soft.ttf'), weight: 700 },
    ],
  },
];
