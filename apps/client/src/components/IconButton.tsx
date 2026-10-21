import { Pressable, type PressableProps } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius } from '@/theme/tokens';

import { Icon, type IconName } from './Icon';

type IconButtonProps = Omit<PressableProps, 'children' | 'style'> & {
  icon: IconName;
  /** Read by screen readers, since there is no visible text. */
  label: string;
  /** Accent tint, for the main action of a screen. */
  accent?: boolean;
};

/** A round quiet button with just an icon, sized like a small Button. */
export function IconButton({ icon, label, accent, disabled, ...props }: IconButtonProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <Pressable
      role="button"
      aria-label={label}
      aria-disabled={disabled}
      disabled={disabled}
      hitSlop={6}
      {...props}
      style={({ pressed }) => [
        styles.base,
        accent && styles.accent,
        pressed && styles.pressed,
        disabled && styles.disabled,
      ]}
    >
      <Icon name={icon} size={accent ? 22 : 18} color={accent ? colors.accent : colors.textSecondary} />
    </Pressable>
  );
}

const useStyles = createStyles((c) => ({
  base: {
    width: 32,
    height: 32,
    borderRadius: radius.pill,
    backgroundColor: c.fill,
    alignItems: 'center',
    justifyContent: 'center',
  },
  accent: {
    width: 36,
    height: 36,
    backgroundColor: c.accentSoft,
  },
  pressed: {
    opacity: 0.7,
  },
  disabled: {
    opacity: 0.4,
  },
}));
