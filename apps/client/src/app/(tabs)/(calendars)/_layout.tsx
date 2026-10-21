import { Stack } from 'expo-router';

import { useTheme } from '@/theme/ThemeProvider';

// A calendar opened from a link still has the list underneath, so back
// and pressing the tab again lead there.
export const unstable_settings = {
  initialRouteName: 'index',
};

/**
 * The calendars tab is a stack, so a calendar's screens open inside the tab
 * and keep the tab bar or sidebar. Pressing the tab again pops back to the list.
 */
export default function CalendarsLayout() {
  const { colors } = useTheme();
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: colors.background },
      }}
    />
  );
}
