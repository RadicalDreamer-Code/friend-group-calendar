import { useRef, useState, type ComponentRef } from 'react';
import { Platform, Pressable, TextInput, View } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { radius, space, type } from '@/theme/tokens';

import { Text } from './Text';

type CodeInputProps = {
  value: string;
  onChange: (value: string) => void;
  length?: number;
  error?: boolean;
  editable?: boolean;
  autoFocus?: boolean;
};

/**
 * One box per digit, backed by a single invisible text field. That keeps
 * pasting, deleting and one-time-code autofill (iOS reads codes from Mail
 * and Messages, browsers offer them too) working like a normal input.
 */
export function CodeInput({ value, onChange, length = 6, error, editable = true, autoFocus }: CodeInputProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const input = useRef<ComponentRef<typeof TextInput>>(null);
  const [focused, setFocused] = useState(false);
  const active = Math.min(value.length, length - 1);

  return (
    <Pressable onPress={() => input.current?.focus()} style={styles.root} accessible={false}>
      {Array.from({ length }, (_, i) => (
        <View key={i} style={[styles.box, focused && i === active && styles.boxActive, error && styles.boxError]}>
          <Text variant="title" style={{ fontFamily: type.headline.fontFamily }}>
            {value[i] ?? ''}
          </Text>
        </View>
      ))}
      <TextInput
        ref={input}
        value={value}
        onChangeText={(text) => onChange(text.replace(/\D/g, '').slice(0, length))}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        editable={editable}
        autoFocus={autoFocus}
        inputMode="numeric"
        keyboardType="number-pad"
        textContentType="oneTimeCode"
        autoComplete={Platform.OS === 'android' ? 'sms-otp' : 'one-time-code'}
        maxLength={length + 2}
        caretHidden
        selectionColor="transparent"
        aria-label="Sign-in code"
        style={[styles.hiddenInput, { color: colors.background }]}
      />
    </Pressable>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    flexDirection: 'row',
    gap: space.sm,
    justifyContent: 'center',
  },
  box: {
    flex: 1,
    maxWidth: 52,
    aspectRatio: 0.82,
    borderRadius: radius.md,
    backgroundColor: c.surface,
    borderWidth: 2,
    borderColor: 'transparent',
    alignItems: 'center',
    justifyContent: 'center',
  },
  boxActive: {
    borderColor: c.accent,
  },
  boxError: {
    borderColor: c.danger,
  },
  // Covers the boxes so taps and long-press-to-paste land on the real input.
  hiddenInput: {
    ...type.title,
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    opacity: 0.011,
    textAlign: 'center',
    outlineWidth: 0,
  },
}));
