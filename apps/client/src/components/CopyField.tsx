import * as Clipboard from 'expo-clipboard';
import { useEffect, useState } from 'react';
import { Platform, View } from 'react-native';

import { haptics } from '@/lib/haptics';
import { createStyles } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';

import { Button } from './Button';
import { Text } from './Text';

const mono = Platform.select({
  ios: 'Menlo',
  android: 'monospace',
  default: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
});

type CopyFieldProps = {
  label: string;
  value: string;
};

/** A read-only value in a monospace box, with a button that copies it. */
export function CopyField({ label, value }: CopyFieldProps) {
  const styles = useStyles();
  const [status, setStatus] = useState<'idle' | 'copied' | 'failed'>('idle');
  const copied = status === 'copied';

  useEffect(() => {
    if (!copied) return;
    const id = setTimeout(() => setStatus('idle'), 2000);
    return () => clearTimeout(id);
  }, [copied]);

  const copy = async () => {
    try {
      // Browsers can refuse, e.g. without clipboard permission.
      if (!(await Clipboard.setStringAsync(value))) throw new Error('not copied');
      haptics.success();
      setStatus('copied');
    } catch {
      setStatus('failed');
    }
  };

  return (
    <View style={styles.root}>
      <View style={styles.header}>
        <Text variant="footnote" tone="secondary">
          {label}
        </Text>
        <Button
          variant="secondary"
          size="small"
          icon={copied ? 'checkmark' : 'copy-outline'}
          title={copied ? 'Copied' : 'Copy'}
          aria-label={copied ? `${label} copied` : `Copy ${label}`}
          onPress={copy}
        />
      </View>
      <View style={styles.box}>
        <Text selectable style={styles.value}>
          {value}
        </Text>
      </View>
      {status === 'failed' ? (
        <Text variant="footnote" tone="danger" aria-live="polite">
          Couldn’t copy. Select the text and copy it yourself.
        </Text>
      ) : (
        <Text variant="footnote" style={styles.srOnly} aria-live="polite">
          {copied ? 'Copied' : ''}
        </Text>
      )}
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    gap: space.sm,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: space.md,
  },
  box: {
    backgroundColor: c.fill,
    borderRadius: radius.sm,
    paddingHorizontal: space.md,
    paddingVertical: space.md,
  },
  value: {
    fontFamily: mono,
    fontSize: 13,
    lineHeight: 19,
    color: c.text,
    // Tokens and commands have no spaces to wrap at.
    ...(Platform.OS === 'web' ? { wordBreak: 'break-all' } : null),
  },
  srOnly: {
    position: 'absolute',
    width: 1,
    height: 1,
    overflow: 'hidden',
    opacity: 0,
  },
}));
