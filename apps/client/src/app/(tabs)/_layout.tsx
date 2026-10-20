import { TabList, Tabs, TabSlot, TabTrigger } from 'expo-router/ui';
import { useWindowDimensions, View } from 'react-native';

import { Sidebar } from '@/components/navigation/Sidebar';
import { TabBar } from '@/components/navigation/TabBar';
import { appTabs } from '@/components/navigation/tabs';
import { createStyles } from '@/theme/ThemeProvider';
import { layout } from '@/theme/tokens';

/**
 * Main navigation. The hidden TabList declares the routes; the visible
 * navigation is a bottom tab bar on phones and a sidebar from 768px.
 */
export default function TabsLayout() {
  const styles = useStyles();
  const { width } = useWindowDimensions();
  const wide = width >= layout.wideBreakpoint;

  return (
    <Tabs style={styles.root}>
      <TabList style={styles.hidden}>
        {appTabs.map((tab) => (
          <TabTrigger key={tab.name} name={tab.name} href={tab.href} />
        ))}
      </TabList>
      <View style={[styles.frame, wide && styles.frameWide]}>
        {wide ? <Sidebar /> : null}
        <View style={styles.content}>
          <TabSlot style={styles.slot} />
        </View>
        {wide ? null : <TabBar />}
      </View>
    </Tabs>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    flex: 1,
    backgroundColor: c.background,
  },
  hidden: {
    display: 'none',
  },
  frame: {
    flex: 1,
  },
  frameWide: {
    flexDirection: 'row',
  },
  content: {
    flex: 1,
  },
  slot: {
    flex: 1,
  },
}));
