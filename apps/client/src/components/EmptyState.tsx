import type { ReactNode } from 'react';
import { View } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon, type IconName } from './Icon';
import { Text } from './Text';

type EmptyStateProps = {
  icon: IconName;
  title: string;
  message: string;
  /** A button below the message, e.g. to create the first item. */
  action?: ReactNode;
  /** Less padding, for a state inside a screen rather than filling it. */
  compact?: boolean;
  /** danger: for errors; the badge turns red and screen readers announce it. */
  tone?: 'default' | 'danger';
};

export function EmptyState({ icon, title, message, action, compact, tone = 'default' }: EmptyStateProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const danger = tone === 'danger';
  return (
    <View style={[styles.root, compact && styles.rootCompact]} role={danger ? 'alert' : undefined}>
      <View style={[styles.badge, danger && styles.badgeDanger]}>
        <Icon name={icon} size={30} color={danger ? colors.danger : colors.accent} />
      </View>
      <Text variant="title" style={styles.title}>
        {title}
      </Text>
      <Text variant="subhead" tone="secondary" style={styles.message}>
        {message}
      </Text>
      {action ? <View style={styles.action}>{action}</View> : null}
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    alignItems: 'center',
    paddingVertical: space.xxxl * 1.5,
    paddingHorizontal: space.xl,
  },
  rootCompact: {
    paddingVertical: space.xxl,
  },
  badge: {
    width: 72,
    height: 72,
    borderRadius: radius.pill,
    backgroundColor: c.accentSoft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: space.lg,
  },
  badgeDanger: {
    backgroundColor: c.fill,
  },
  title: {
    textAlign: 'center',
    marginBottom: space.sm,
  },
  message: {
    textAlign: 'center',
    maxWidth: 300,
  },
  action: {
    marginTop: space.xl,
  },
}));
