import type { PressableProps } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius } from '@/theme/tokens';

import { Icon, type IconName } from './Icon';
import { Springy } from './Springy';

type IconButtonProps = Omit<PressableProps, 'children' | 'style'> & {
  icon: IconName;
  /** Read by screen readers, since there is no visible text. */
  label: string;
  /** Filled with ink, for the main action of a screen. */
  accent?: boolean;
};

/** A round button with just an icon on a soft fill, springy when pressed. */
export function IconButton({ icon, label, accent, disabled, ...props }: IconButtonProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <Springy
      role="button"
      aria-label={label}
      aria-disabled={disabled}
      disabled={disabled}
      hitSlop={6}
      scaleTo={0.88}
      {...props}
      style={({ pressed }) => [
        styles.base,
        accent && styles.accent,
        pressed && styles.pressed,
        disabled && styles.disabled,
      ]}
    >
      <Icon name={icon} size={accent ? 22 : 18} color={accent ? colors.background : colors.text} />
    </Springy>
  );
}

const SIZE = 38;

const useStyles = createStyles((c) => ({
  base: {
    width: SIZE,
    height: SIZE,
    borderRadius: radius.pill,
    backgroundColor: c.surface,
    alignItems: 'center',
    justifyContent: 'center',
  },
  accent: {
    backgroundColor: c.text,
  },
  pressed: {
    opacity: 0.85,
  },
  disabled: {
    opacity: 0.35,
  },
}));
