import { Platform, Switch as RNSwitch, type SwitchProps as RNSwitchProps } from 'react-native';

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
export function Switch({ label, ...props }: SwitchProps) {
  const { colors } = useTheme();
  return (
    <RNSwitch
      aria-label={label}
      trackColor={{ false: colors.fill, true: colors.accent }}
      thumbColor={THUMB}
      ios_backgroundColor={colors.fill}
      {...webThumb}
      {...props}
    />
  );
}
