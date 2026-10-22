import { Platform, Switch as RNSwitch, type SwitchProps as RNSwitchProps } from 'react-native';

import { haptics } from '@/lib/haptics';
import { useTheme } from '@/theme/ThemeProvider';

type SwitchProps = Omit<RNSwitchProps, 'trackColor' | 'thumbColor'> & {
  /** Read by screen readers; the visible label sits next to the switch. */
  label: string;
};

const THUMB = '#FFFFFF';
// react-native-web colors the thumb of a switch that is on with its own
// activeThumbColor (teal by default), which React Native's types do not know.
const webThumb = Platform.OS === 'web' ? ({ activeThumbColor: THUMB } as object) : {};

/** The platform switch in the accent color. */
export function Switch({ label, onValueChange, ...props }: SwitchProps) {
  const { colors } = useTheme();
  return (
    <RNSwitch
      aria-label={label}
      // The off track needs 3:1 against the row to be seen at all; the pale fill was close to invisible.
      trackColor={{ false: colors.textTertiary, true: colors.accent }}
      thumbColor={THUMB}
      ios_backgroundColor={colors.textTertiary}
      {...webThumb}
      {...props}
      onValueChange={(value) => {
        haptics.selection();
        onValueChange?.(value);
      }}
    />
  );
}
