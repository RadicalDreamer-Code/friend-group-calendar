import { Image, View } from 'react-native';

import { createStyles } from '@/theme/ThemeProvider';
import { entryColors, radius } from '@/theme/tokens';

import { Text } from './Text';

type AvatarProps = {
  /** Used to pick a stable background color for the initials. */
  id: string;
  name: string;
  url?: string | null;
  size?: number;
};

/** A round photo, or the person's initials on a muted color when there is none. */
export function Avatar({ id, name, url, size = 40 }: AvatarProps) {
  const styles = useStyles();
  const box = { width: size, height: size, borderRadius: radius.pill };

  if (url) {
    return <Image source={{ uri: url }} style={[styles.image, box]} accessibilityIgnoresInvertColors alt="" />;
  }
  return (
    <View style={[styles.fallback, box, { backgroundColor: colorFor(id) }]} aria-hidden>
      <Text style={[styles.initials, { fontSize: size * 0.4, lineHeight: size * 0.5 }]}>{initials(name)}</Text>
    </View>
  );
}

export function initials(name: string): string {
  const parts = name
    .replace(/@.*/, '')
    .split(/[\s._-]+/)
    .filter(Boolean);
  const letters = parts.length > 1 ? parts[0][0] + parts[1][0] : (parts[0] ?? '?').slice(0, 2);
  return letters.toUpperCase();
}

function colorFor(id: string): string {
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = (hash * 31 + id.charCodeAt(i)) | 0;
  return entryColors[Math.abs(hash) % entryColors.length].value;
}

const useStyles = createStyles((c) => ({
  image: {
    backgroundColor: c.fill,
  },
  fallback: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  initials: {
    color: '#FFFFFF',
    fontWeight: '600',
  },
}));
