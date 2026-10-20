import { View } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon, type IconName } from './Icon';
import { Text } from './Text';

type EmptyStateProps = {
  icon: IconName;
  title: string;
  message: string;
};

export function EmptyState({ icon, title, message }: EmptyStateProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <View style={styles.root}>
      <View style={styles.badge}>
        <Icon name={icon} size={26} color={colors.accent} />
      </View>
      <Text variant="headline" style={styles.title}>
        {title}
      </Text>
      <Text variant="subhead" tone="secondary" style={styles.message}>
        {message}
      </Text>
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    alignItems: 'center',
    paddingVertical: space.xxxl * 1.5,
    paddingHorizontal: space.xl,
  },
  badge: {
    width: 56,
    height: 56,
    borderRadius: radius.pill,
    backgroundColor: c.accentSoft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: space.lg,
  },
  title: {
    textAlign: 'center',
    marginBottom: space.xs,
  },
  message: {
    textAlign: 'center',
    maxWidth: 300,
  },
}));
