import type { TabTriggerSlotProps } from 'expo-router/ui';
import { Pressable, type PressableStateCallbackType } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon } from '../Icon';
import { Text } from '../Text';
import type { AppTab } from './tabs';

type NavItemProps = TabTriggerSlotProps & {
  tab: AppTab;
  variant: 'bar' | 'sidebar';
};

/** Receives `isFocused` and press handlers from `<TabTrigger asChild>`. */
export function NavItem({ tab, variant, isFocused = false, style: _style, ref, ...pressableProps }: NavItemProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const tint = isFocused ? colors.accent : variant === 'bar' ? colors.textTertiary : colors.textSecondary;

  const pressableStyle = ({ pressed, hovered }: PressableStateCallbackType & { hovered?: boolean }) => [
    variant === 'bar' ? styles.bar : styles.sidebar,
    variant === 'sidebar' && isFocused && styles.sidebarFocused,
    variant === 'sidebar' && !isFocused && hovered && styles.sidebarHovered,
    pressed && styles.pressed,
  ];

  return (
    <Pressable
      {...pressableProps}
      // TabTrigger's ref type targets the View class; Pressable renders a View, so this is safe.
      ref={ref as never}
      role="tab"
      aria-selected={isFocused}
      aria-label={tab.label}
      style={pressableStyle}
    >
      <Icon name={isFocused ? tab.iconFocused : tab.icon} size={variant === 'bar' ? 24 : 20} color={tint} />
      <Text
        variant={variant === 'bar' ? 'caption' : 'callout'}
        style={[{ color: tint }, variant === 'sidebar' && isFocused && styles.sidebarLabelFocused]}
      >
        {tab.label}
      </Text>
    </Pressable>
  );
}

const useStyles = createStyles((c) => ({
  bar: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: space.xxs,
    paddingTop: space.xs,
  },
  sidebar: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    height: 40,
    paddingHorizontal: space.md,
    borderRadius: radius.sm,
  },
  sidebarFocused: {
    backgroundColor: c.accentSoft,
  },
  sidebarHovered: {
    backgroundColor: c.fill,
  },
  sidebarLabelFocused: {
    fontWeight: '600',
  },
  pressed: {
    opacity: 0.6,
  },
}));
