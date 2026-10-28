import { Stack } from 'expo-router';
import { useReducedMotion } from 'react-native-reanimated';

import { useTheme } from '@/theme/ThemeProvider';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

// A profile screen opened from a link still has Profile underneath.
export const unstable_settings = {
  anchor: 'profile/index',
};

/** The profile tab is a stack, so its screens keep the tab bar or sidebar, like the calendars tab. */
export default function ProfileLayout() {
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
