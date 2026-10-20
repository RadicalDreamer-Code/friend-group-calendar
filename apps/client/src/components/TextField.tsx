import { useState, type ComponentRef, type Ref } from 'react';
import { TextInput, View, type TextInputProps } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { hairline, radius, space, type } from '@/theme/tokens';

import { Text } from './Text';

type TextFieldProps = TextInputProps & {
  label: string;
  /** Shown below the field; red when `error` is set. */
  hint?: string;
  /** Green hint, e.g. for "Available". */
  hintSuccess?: boolean;
  error?: string | null;
  ref?: Ref<ComponentRef<typeof TextInput>>;
};

export function TextField({ label, hint, hintSuccess, error, style, onFocus, onBlur, ref, ...props }: TextFieldProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const message = error ?? hint;

  return (
    <View style={styles.root}>
      <Text variant="footnote" tone="secondary" style={styles.label}>
        {label}
      </Text>
      <TextInput
        ref={ref}
        aria-label={label}
        aria-invalid={!!error}
        placeholderTextColor={colors.textTertiary}
        selectionColor={colors.accent}
        {...props}
        onFocus={(e) => {
          setFocused(true);
          onFocus?.(e);
        }}
        onBlur={(e) => {
          setFocused(false);
          onBlur?.(e);
        }}
        style={[styles.input, focused && styles.inputFocused, !!error && styles.inputError, style]}
      />
      {message ? (
        <Text
          variant="footnote"
          tone={error ? 'danger' : hintSuccess ? 'success' : 'tertiary'}
          style={styles.message}
          aria-live="polite"
        >
          {message}
        </Text>
      ) : null}
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    gap: space.xs,
  },
  label: {
    paddingHorizontal: space.xs,
  },
  input: {
    ...type.body,
    color: c.text,
    height: 50,
    paddingHorizontal: space.lg,
    borderRadius: radius.md,
    backgroundColor: c.surface,
    borderWidth: hairline,
    borderColor: c.hairline,
    // Web: the border change below is the focus indicator.
    outlineWidth: 0,
  },
  inputFocused: {
    borderColor: c.accent,
    borderWidth: 1.5,
  },
  inputError: {
    borderColor: c.danger,
  },
  message: {
    paddingHorizontal: space.xs,
  },
}));
