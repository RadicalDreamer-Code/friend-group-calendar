import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useFonts } from 'expo-font';
import { SplashScreen, Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useEffect } from 'react';
import { View } from 'react-native';
import { useReducedMotion } from 'react-native-reanimated';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { SessionProvider, useSession } from '@/auth/session';
import { SignInFlowProvider } from '@/auth/signInFlow';
import { ErrorState } from '@/components/ErrorState';
import { listenForAppFocus } from '@/lib/freshness';
import { appFonts } from '@/theme/fonts';
import { ThemeProvider, useTheme } from '@/theme/ThemeProvider';

SplashScreen.preventAutoHideAsync();
listenForAppFocus();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1 },
  },
});

export default function RootLayout() {
  // The splash screen stays up until the fonts are in. If they fail, text
  // falls back to the system font rather than keeping the app from starting.
  const [fontsLoaded, fontError] = useFonts(appFonts);
  useEffect(() => {
    if (fontError) console.warn('Fonts failed to load, using the system font.', fontError);
  }, [fontError]);
  if (!fontsLoaded && !fontError) return null;

  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <SessionProvider>
            <SignInFlowProvider>
              <RootNavigator />
            </SignInFlowProvider>
          </SessionProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

/**
 * Which screens exist depends on the session. Signed-out users only reach
 * the sign-in flow, new users only the username picker, everyone else the
 * app; Expo Router redirects away from screens whose guard is false.
 */
function RootNavigator() {
  const { scheme, colors } = useTheme();
  const { status, error, retry, retrying } = useSession();
  // Screens appear in place for people who prefer reduced motion.
  const reduceMotion = useReducedMotion();
  const sheet = { presentation: 'modal', animation: reduceMotion ? 'none' : 'slide_from_bottom' } as const;

  useEffect(() => {
    if (status !== 'loading') SplashScreen.hide();
  }, [status]);

  // Wait for the session check before mounting any screen, so a deep link
  // like /profile is not redirected to sign-in while we still find out.
  if (status === 'loading') return <View style={{ flex: 1, backgroundColor: colors.background }} />;
  if (status === 'unreachable') {
    return (
      <View style={{ flex: 1, justifyContent: 'center', backgroundColor: colors.background }}>
        <ErrorState error={error} onRetry={retry} retrying={retrying} />
      </View>
    );
  }

  return (
    <>
      <StatusBar style={scheme === 'dark' ? 'light' : 'dark'} />
      <Stack
        screenOptions={{
          headerShown: false,
          animation: reduceMotion ? 'none' : 'fade',
          contentStyle: { backgroundColor: colors.background },
        }}
      >
        <Stack.Protected guard={status === 'signedIn'}>
          <Stack.Screen name="(tabs)" />
          <Stack.Screen name="edit-username" options={sheet} />
          <Stack.Screen name="new-calendar" options={sheet} />
          <Stack.Screen name="edit-entry" options={sheet} />
        </Stack.Protected>
        <Stack.Protected guard={status === 'needsUsername'}>
          <Stack.Screen name="welcome" />
        </Stack.Protected>
        <Stack.Protected guard={status === 'signedOut'}>
          <Stack.Screen name="sign-in" />
          <Stack.Screen name="verify" options={{ animation: reduceMotion ? 'none' : 'slide_from_right' }} />
        </Stack.Protected>
      </Stack>
    </>
  );
}
