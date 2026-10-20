import type { TabTriggerSlotProps } from 'expo-router/ui';
import { Pressable, View, type PressableStateCallbackType, type StyleProp, type ViewStyle } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Icon } from '../Icon';
import { Text } from '../Text';
import type { AppTab } from './tabs';

type NavItemProps = TabTriggerSlotProps & {
  tab: AppTab;
  variant: 'bar' | 'sidebar';
  /** Count of things waiting for the user, e.g. friend requests; hidden at 0. */
  badge?: number;
};

/** Receives `isFocused` and press handlers from `<TabTrigger asChild>`. */
export function NavItem({
  tab,
  variant,
  badge = 0,
  isFocused = false,
  style: _style,
  ref,
  ...pressableProps
}: NavItemProps) {
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
      aria-label={badge > 0 ? `${tab.label}, ${badge} new` : tab.label}
      style={pressableStyle}
    >
      <View>
        <Icon name={isFocused ? tab.iconFocused : tab.icon} size={variant === 'bar' ? 24 : 20} color={tint} />
        {badge > 0 && variant === 'bar' ? <Badge count={badge} style={styles.barBadge} /> : null}
      </View>
      <Text
        variant={variant === 'bar' ? 'caption' : 'callout'}
        style={[
          { color: tint },
          variant === 'sidebar' && styles.sidebarLabel,
          variant === 'sidebar' && isFocused && styles.sidebarLabelFocused,
        ]}
      >
        {tab.label}
      </Text>
      {badge > 0 && variant === 'sidebar' ? <Badge count={badge} /> : null}
    </Pressable>
  );
}

function Badge({ count, style }: { count: number; style?: StyleProp<ViewStyle> }) {
  const styles = useStyles();
  return (
    <View style={[styles.badge, style]} aria-hidden>
      <Text variant="caption" tone="onAccent" style={styles.badgeText}>
        {count > 9 ? '9+' : count}
      </Text>
    </View>
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
  sidebarLabel: {
    flex: 1,
  },
  sidebarLabelFocused: {
    fontWeight: '600',
  },
  pressed: {
    opacity: 0.6,
  },
  badge: {
    minWidth: 18,
    height: 18,
    paddingHorizontal: space.xs,
    borderRadius: radius.pill,
    backgroundColor: c.accent,
    alignItems: 'center',
    justifyContent: 'center',
  },
  barBadge: {
    position: 'absolute',
    top: -4,
    left: 16,
    // Separates the badge from the icon underneath.
    borderWidth: 2,
    borderColor: c.background,
    height: 22,
    minWidth: 22,
  },
  badgeText: {
    fontWeight: '700',
  },
}));
