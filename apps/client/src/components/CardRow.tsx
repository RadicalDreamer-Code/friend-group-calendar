import type { ComponentProps, ReactNode } from 'react';
import type { StyleProp, ViewStyle } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon } from './Icon';
import { Springy } from './Springy';

type CardRowProps = Omit<ComponentProps<typeof Springy>, 'style' | 'children' | 'scaleTo'> & {
  /** Padding and the like; the card's look comes from here. */
  style?: StyleProp<ViewStyle>;
  children: ReactNode;
};

/**
 * A soft card that opens something: its content in a row with a chevron at
 * the end, springy and a shade darker while pressed or hovered.
 */
export function CardRow({ style, children, ...props }: CardRowProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <Springy
      {...props}
      scaleTo={0.98}
      style={({ pressed, hovered }: { pressed: boolean; hovered?: boolean }) => [
        styles.card,
        style,
        (pressed || hovered) && styles.pressed,
      ]}
    >
      {children}
      <Icon name="chevron-forward" size={18} color={colors.textTertiary} />
    </Springy>
  );
}

const useStyles = createStyles((c) => ({
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    backgroundColor: c.surface,
    borderRadius: radius.lg,
    paddingRight: space.md,
  },
  pressed: {
    backgroundColor: c.fill,
  },
}));
