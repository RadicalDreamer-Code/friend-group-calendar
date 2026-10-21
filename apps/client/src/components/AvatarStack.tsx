import { View } from 'react-native';

import type { Person } from '@/api/client';
import { createStyles } from '@/theme/ThemeProvider';
import { radius } from '@/theme/tokens';

import { Avatar } from './Avatar';
import { Text } from './Text';

type AvatarStackProps = {
  people: Person[];
  /** Everyone in the group; more than `people` adds a "+N" bubble. */
  total: number;
  size?: number;
};

const RING = 2;

/** Overlapping avatars of a group, e.g. a calendar's members. */
export function AvatarStack({ people, total, size = 28 }: AvatarStackProps) {
  const styles = useStyles();
  const more = total - people.length;
  const outer = size + RING * 2;
  const ring = { width: outer, height: outer, borderRadius: radius.pill, marginLeft: -size * 0.3 };

  return (
    <View style={styles.root} aria-hidden>
      {people.map((p, i) => (
        <View key={p.id} style={[styles.ring, ring, i === 0 && styles.first]}>
          <Avatar id={p.id} name={p.username} url={p.avatarUrl} size={size} />
        </View>
      ))}
      {more > 0 ? (
        <View style={[styles.ring, ring, people.length === 0 && styles.first]}>
          <View style={[styles.more, { width: size, height: size }]}>
            <Text variant="caption" tone="secondary" style={styles.moreText}>
              +{more}
            </Text>
          </View>
        </View>
      ) : null}
    </View>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  ring: {
    padding: RING,
    backgroundColor: c.surface,
  },
  first: {
    marginLeft: 0,
  },
  more: {
    borderRadius: radius.pill,
    backgroundColor: c.fill,
    alignItems: 'center',
    justifyContent: 'center',
  },
  moreText: {
    fontWeight: '600',
  },
}));
