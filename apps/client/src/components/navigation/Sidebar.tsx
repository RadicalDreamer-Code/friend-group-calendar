import { TabTrigger } from 'expo-router/ui';
import { View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useIncomingRequestCount } from '@/api/friends';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { hairline, layout, radius, space } from '@/theme/tokens';

import { Icon } from '../Icon';
import { Text } from '../Text';
import { NavItem } from './NavItem';
import { appTabs } from './tabs';

/** Sidebar navigation for tablet and desktop widths. */
export function Sidebar() {
  const requests = useIncomingRequestCount();
  const styles = useStyles();
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <View style={[styles.root, { paddingTop: insets.top + space.xl }]}>
      <View style={styles.brand}>
        <View style={styles.logo}>
          <Icon name="calendar" size={16} color={colors.onAccent} />
        </View>
        <Text variant="headline">Friend Calendar</Text>
      </View>
      <View role="tablist" aria-orientation="vertical" style={styles.items}>
        {appTabs.map((tab) => (
          <TabTrigger key={tab.name} name={tab.name} asChild>
            <NavItem tab={tab} variant="sidebar" badge={tab.name === 'friends' ? requests : 0} />
          </TabTrigger>
        ))}
      </View>
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    width: layout.sidebarWidth,
    paddingHorizontal: space.md,
    backgroundColor: c.background,
    borderRightWidth: hairline,
    borderRightColor: c.hairline,
  },
  brand: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
    paddingHorizontal: space.md,
    marginBottom: space.xl,
  },
  logo: {
    width: 28,
    height: 28,
    borderRadius: radius.sm,
    backgroundColor: c.accent,
    alignItems: 'center',
    justifyContent: 'center',
  },
  items: {
    gap: space.xxs,
  },
}));
