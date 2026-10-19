import { Text as RNText, type TextProps as RNTextProps } from 'react-native';

import { useTheme } from '@/theme/ThemeProvider';
import { type as typeScale, type Colors, type TypeVariant } from '@/theme/tokens';

type Tone = 'primary' | 'secondary' | 'tertiary' | 'accent' | 'danger' | 'success' | 'onAccent';

const toneColor: Record<Tone, keyof Colors> = {
  primary: 'text',
  secondary: 'textSecondary',
  tertiary: 'textTertiary',
  accent: 'accent',
  danger: 'danger',
  success: 'success',
  onAccent: 'onAccent',
};

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  tone?: Tone;
};

export function Text({ variant = 'body', tone = 'primary', style, ...props }: TextProps) {
  const { colors } = useTheme();
  return <RNText {...props} style={[typeScale[variant], { color: colors[toneColor[tone]] }, style]} />;
}
