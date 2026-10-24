import { Stack } from 'expo-router';
import { useReducedMotion } from 'react-native-reanimated';

import { useTheme } from '@/theme/ThemeProvider';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

// A calendar opened from a link still has the list underneath, so back
// and pressing the tab again lead there.
export const unstable_settings = {
  anchor: 'index',
};

/**
 * The calendars tab is a stack, so a calendar's screens open inside the tab
 * and keep the tab bar or sidebar. Pressing the tab again pops back to the list.
 */
export default function CalendarsLayout() {
  const { colors } = useTheme();
  const reduceMotion = useReducedMotion();
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        animation: reduceMotion ? 'none' : 'default',
        contentStyle: { backgroundColor: colors.background },
      }}
    />
  );
}
