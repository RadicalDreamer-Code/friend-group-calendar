import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon } from './Icon';
import { RainbowArc } from './RainbowArc';
import { Text } from './Text';

type AuthScreenProps = {
  title: string;
  subtitle: ReactNode;
  children: ReactNode;
};

/** The calm, centered layout shared by the sign-in steps, with a sunset peeking in from the corner. */
export function AuthScreen({ title, subtitle, children }: AuthScreenProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();

  return (
    <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <RainbowArc size={ARC} style={[styles.arc, { top: insets.top - ARC / 2.6 }]} />
      <ScrollView
        contentContainerStyle={[
          styles.scroll,
          { paddingTop: insets.top + space.xxxl, paddingBottom: insets.bottom + space.xl },
        ]}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.inner}>
          <View style={styles.logo}>
            <Icon name="calendar" size={28} color={colors.onAccent} />
          </View>
          <Text variant="largeTitle" role="heading" aria-level={1} style={styles.title}>
            {title}
          </Text>
          <Text variant="callout" tone="secondary" style={styles.subtitle}>
            {subtitle}
          </Text>
          <View style={styles.body}>{children}</View>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const ARC = 260;

const useStyles = createStyles((c) => ({
  root: {
    flex: 1,
    backgroundColor: c.background,
    overflow: 'hidden',
  },
  arc: {
    position: 'absolute',
    right: -ARC / 2.6,
  },
  scroll: {
    flexGrow: 1,
    justifyContent: 'center',
    paddingHorizontal: space.xl,
  },
  inner: {
    width: '100%',
    maxWidth: 400,
    alignSelf: 'center',
  },
  logo: {
    width: 60,
    height: 60,
    borderRadius: radius.lg,
    backgroundColor: c.accent,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: space.xl,
  },
  title: {
    marginBottom: space.sm,
  },
  subtitle: {
    marginBottom: space.xxl,
  },
  body: {
    gap: space.lg,
  },
}));
