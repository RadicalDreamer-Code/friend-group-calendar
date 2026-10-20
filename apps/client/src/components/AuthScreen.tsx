import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon } from './Icon';
import { Text } from './Text';

type AuthScreenProps = {
  title: string;
  subtitle: ReactNode;
  children: ReactNode;
};

/** The calm, centered layout shared by the sign-in steps. */
export function AuthScreen({ title, subtitle, children }: AuthScreenProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();

  return (
    <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScrollView
        contentContainerStyle={[
          styles.scroll,
          { paddingTop: insets.top + space.xxxl, paddingBottom: insets.bottom + space.xl },
        ]}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.inner}>
          <View style={styles.logo}>
            <Icon name="calendar" size={26} color={colors.onAccent} />
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

const useStyles = createStyles((c) => ({
  root: {
    flex: 1,
    backgroundColor: c.background,
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
    width: 52,
    height: 52,
    borderRadius: radius.md,
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
