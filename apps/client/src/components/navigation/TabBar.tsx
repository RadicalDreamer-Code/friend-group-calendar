import { TabTrigger } from 'expo-router/ui';
import { View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { createStyles } from '@/theme/ThemeProvider';
import { hairline, layout } from '@/theme/tokens';

import { NavItem } from './NavItem';
import { appTabs } from './tabs';

/** Bottom tab bar for phone-sized screens. */
export function TabBar() {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  return (
    <View
      role="tablist"
      style={[styles.root, { paddingBottom: insets.bottom, height: layout.tabBarHeight + insets.bottom }]}
    >
      {appTabs.map((tab) => (
        <TabTrigger key={tab.name} name={tab.name} asChild>
          <NavItem tab={tab} variant="bar" />
        </TabTrigger>
      ))}
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    flexDirection: 'row',
    backgroundColor: c.background,
    borderTopWidth: hairline,
    borderTopColor: c.hairline,
  },
}));
