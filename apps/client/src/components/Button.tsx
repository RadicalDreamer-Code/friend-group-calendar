import { ActivityIndicator, Pressable, type PressableProps } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Text } from './Text';

type ButtonProps = Omit<PressableProps, 'children' | 'style'> & {
  title: string;
  /** primary: filled accent. secondary: quiet fill. plain: text only. */
  variant?: 'primary' | 'secondary' | 'plain';
  destructive?: boolean;
  loading?: boolean;
  /** small: a compact pill for actions inside list rows. */
  size?: 'regular' | 'small';
};

export function Button({
  title,
  variant = 'primary',
  destructive,
  loading,
  disabled,
  size = 'regular',
  ...props
}: ButtonProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const inactive = disabled || loading;
  const textColor =
    variant === 'primary'
      ? colors.onAccent
      : destructive
        ? colors.danger
        : variant === 'plain'
          ? colors.accent
          : colors.text;

  return (
    <Pressable
      role="button"
      // The title is replaced by a spinner while loading; the name stays.
      aria-label={title}
      aria-disabled={inactive}
      aria-busy={loading}
      disabled={inactive}
      {...props}
      style={({ pressed }) => [
        styles.base,
        variant === 'primary' && [styles.primary, destructive && styles.primaryDestructive],
        variant === 'secondary' && styles.secondary,
        variant === 'plain' && styles.plain,
        size === 'small' && styles.small,
        pressed && styles.pressed,
        inactive && !loading && styles.disabled,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={textColor} />
      ) : (
        <Text
          variant={size === 'small' ? 'subhead' : 'headline'}
          style={[{ color: textColor }, size === 'small' && styles.smallText]}
        >
          {title}
        </Text>
      )}
    </Pressable>
  );
}

const useStyles = createStyles((c) => ({
  base: {
    height: 50,
    borderRadius: radius.md,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: space.xl,
  },
  primary: {
    backgroundColor: c.accent,
  },
  primaryDestructive: {
    backgroundColor: c.danger,
  },
  secondary: {
    backgroundColor: c.fill,
  },
  plain: {
    height: 44,
    paddingHorizontal: space.md,
  },
  small: {
    height: 32,
    borderRadius: radius.pill,
    paddingHorizontal: space.md,
  },
  smallText: {
    fontWeight: '600',
  },
  pressed: {
    opacity: 0.7,
  },
  disabled: {
    opacity: 0.4,
  },
}));
