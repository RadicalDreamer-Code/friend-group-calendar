import { router, useLocalSearchParams } from 'expo-router';
import { useEffect, useRef, useState } from 'react';
import { ScrollView, useWindowDimensions, View } from 'react-native';
import Animated, {
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withSequence,
  withSpring,
  withTiming,
} from 'react-native-reanimated';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { canEditEntries, useCalendar } from '@/api/calendars';
import { errorMessage, type Calendar, type CalendarMember, type Entry, type EntryCover } from '@/api/client';
import { isEntryGone, replyOf, useEntry, useRespond, type Reply } from '@/api/entries';
import { useUser } from '@/auth/session';
import { Avatar } from '@/components/Avatar';
import { Button } from '@/components/Button';
import { CalendarError } from '@/components/CalendarError';
import { Cover } from '@/components/Cover';
import { CoverIcon } from '@/components/CoverIcon';
import { EmptyState } from '@/components/EmptyState';
import { RefreshError } from '@/components/ErrorState';
import { Icon } from '@/components/Icon';
import { IconButton } from '@/components/IconButton';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { Text } from '@/components/Text';
import { countdown, formatDay, formatShortDay } from '@/lib/dates';
import { haptics } from '@/lib/haptics';
import { useRefreshOnFocus } from '@/lib/freshness';
import { backTo } from '@/lib/navigation';
import { createStyles } from '@/theme/ThemeProvider';
import { entryColor, entryInk, layout, motion, radius, space } from '@/theme/tokens';

/** One entry: its cover, a countdown, your answer, and who is coming. */
export default function EntryScreen() {
  const { id, entryId } = useLocalSearchParams<{ id: string; entryId: string }>();
  const calendar = useCalendar(id);
  const entry = useEntry(id, entryId);
  useRefreshOnFocus(calendar, entry);
  const back = {
    label: calendar.data?.name ?? 'Calendar',
    onPress: () => backTo({ pathname: '/calendars/[id]', params: entry.data ? { id, day: entry.data.date } : { id } }),
  };

  const retry = () => {
    if (calendar.isError) calendar.refetch();
    if (entry.isError) entry.refetch();
  };
  const retrying = calendar.isFetching || entry.isFetching;

  if (entry.isError && isEntryGone(entry.error)) {
    return (
      <Page title="Entry" back={back}>
        <EmptyState
          icon="calendar-clear-outline"
          title="Entry deleted"
          message="Someone removed it from the calendar."
          action={
            <Button variant="secondary" title={`Back to ${calendar.data?.name ?? 'calendar'}`} onPress={back.onPress} />
          }
        />
      </Page>
    );
  }
  if ((calendar.isError && !calendar.data) || (entry.isError && !entry.data)) {
    return (
      <Page title="Entry" back={back}>
        <CalendarError error={calendar.error ?? entry.error} onRetry={retry} retrying={retrying} />
      </Page>
    );
  }
  if (!calendar.data || !entry.data) {
    return (
      <Page title="Entry" back={back}>
        <SkeletonList title="Who’s coming" rows={2} leading="avatar" />
      </Page>
    );
  }

  return (
    <Details
      calendar={calendar.data}
      entry={entry.data}
      back={back}
      refreshError={calendar.isRefetchError || entry.isRefetchError ? (calendar.error ?? entry.error) : null}
      onRetry={retry}
      retrying={retrying}
    />
  );
}

function Details({
  calendar,
  entry,
  back,
  refreshError,
  onRetry,
  retrying,
}: {
  calendar: Calendar;
  entry: Entry;
  back: { label: string; onPress: () => void };
  refreshError: unknown;
  onRetry: () => void;
  retrying: boolean;
}) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  const { width } = useWindowDimensions();
  const wide = width >= layout.wideBreakpoint;
  const me = useUser();
  const respond = useRespond(calendar.id, entry.id, me.id);
  const mine = replyOf(entry, me.id);
  const canEdit = canEditEntries(calendar.role);

  const byId = new Map(calendar.members.map((m) => [m.user.id, m]));
  const people = (ids: string[]) => ids.map((id) => byId.get(id)).filter((m): m is CalendarMember => !!m);
  const going = people(entry.accepted);
  const declined = people(entry.declined);
  const answered = new Set([...entry.accepted, ...entry.declined]);
  const waiting = calendar.members.filter((m) => !answered.has(m.user.id));
  // Your face moves to another spot when you say you're in, maybe as a new
  // view; this tells it to bounce in rather than just appear.
  const [answerShown, setAnswerShown] = useState(mine);
  const [joined, setJoined] = useState(false);
  if (mine !== answerShown) {
    setAnswerShown(mine);
    setJoined(mine === 'accepted');
  }

  // Tapping your current answer again takes it back.
  const answer = (reply: Exclude<Reply, null>) => {
    haptics.selection();
    respond.mutate(mine === reply ? null : reply);
  };

  const edit = () => router.push({ pathname: '/edit-entry', params: { calendarId: calendar.id, entryId: entry.id } });

  return (
    <View style={styles.root}>
      <ScrollView
        contentContainerStyle={[
          styles.scroll,
          wide && { paddingTop: insets.top + space.xxl, paddingHorizontal: space.xxl },
        ]}
      >
        <View style={styles.inner}>
          <Cover
            color={entryColor(entry.color)}
            cover={entry.cover}
            sunAt={0.66}
            style={[styles.cover, { height: COVER_HEIGHT + (wide ? 0 : insets.top) }, wide && styles.coverWide]}
          >
            <View style={styles.coverText}>
              <View style={styles.chip}>
                <Text variant="caption" tone="onSunset" style={styles.chipText}>
                  {formatShortDay(entry.date)}
                </Text>
              </View>
              <Text variant="largeTitle" role="heading" aria-level={1} style={styles.title} numberOfLines={3}>
                {entry.title}
              </Text>
            </View>
          </Cover>

          <View style={[styles.sheet, wide && styles.sheetWide]}>
            {refreshError ? <RefreshError error={refreshError} onRetry={onRetry} retrying={retrying} /> : null}

            <View style={styles.countdown}>
              <CountdownTile color={entryColor(entry.color)} cover={entry.cover} jig={mine === 'accepted'} />
              <View style={styles.countdownText}>
                <Text variant="title">{countdown(entry.date)}</Text>
                <Text variant="footnote" tone="secondary">
                  {formatDay(entry.date)}
                  {entry.createdBy
                    ? ` · Added by ${entry.createdBy.id === me.id ? 'you' : `@${entry.createdBy.username}`}`
                    : ''}
                </Text>
              </View>
            </View>

            <View style={styles.answers} role="radiogroup" aria-label="Your answer">
              <View style={styles.answerWide}>
                <Button
                  role="radio"
                  aria-checked={mine === 'accepted'}
                  title={mine === 'accepted' ? 'You’re in' : 'I’m in'}
                  icon={mine === 'accepted' ? 'checkmark' : undefined}
                  selected={mine === 'accepted'}
                  onPress={() => answer('accepted')}
                />
              </View>
              <View style={styles.answer}>
                <Button
                  role="radio"
                  aria-checked={mine === 'declined'}
                  variant="secondary"
                  title="Can’t"
                  selected={mine === 'declined'}
                  onPress={() => answer('declined')}
                />
              </View>
            </View>
            <Text variant="footnote" tone={respond.isError ? 'danger' : 'tertiary'} style={styles.answerHint}>
              {respond.isError
                ? errorMessage(respond.error)
                : mine
                  ? 'Tap your answer again to take it back.'
                  : 'Let the others know if you’re in.'}
            </Text>

            <Faces
              title={`Who’s coming · ${going.length} of ${calendar.members.length}`}
              groups={[
                { members: going, status: 'going' },
                { members: waiting, status: 'waiting' },
              ]}
              me={me.id}
              joined={joined}
            />
            {declined.length > 0 ? (
              <Faces
                title={`Can’t make it · ${declined.length}`}
                groups={[{ members: declined, status: 'declined' }]}
                me={me.id}
                joined={false}
              />
            ) : null}
          </View>
        </View>
      </ScrollView>

      {/* Round buttons that stay put over the cover and the sheet. */}
      <View
        style={[styles.floating, { top: (wide ? insets.top + space.xxl : insets.top) + space.md }]}
        pointerEvents="box-none"
      >
        <View style={[styles.floatingRow, wide && styles.floatingRowWide]} pointerEvents="box-none">
          <FloatingButton icon="chevron-back" label={`Back to ${back.label}`} onPress={back.onPress} />
          {canEdit ? <FloatingButton icon="create-outline" label="Edit entry" onPress={edit} /> : null}
        </View>
      </View>
    </View>
  );
}

/** A white round button over the cover. */
function FloatingButton({
  icon,
  label,
  onPress,
}: {
  icon: 'chevron-back' | 'create-outline';
  label: string;
  onPress: () => void;
}) {
  const styles = useStyles();
  return (
    <View style={styles.floatingButton}>
      <IconButton icon={icon} label={label} onPress={onPress} />
    </View>
  );
}

/** The entry's color tile with its cover icon beside the countdown; it does a little jig when you say you're in. */
function CountdownTile({ color, cover, jig }: { color: string; cover: EntryCover | null; jig: boolean }) {
  const styles = useStyles();
  const reduceMotion = useReducedMotion();
  const tilt = useSharedValue(0);
  const lift = useSharedValue(0);
  const first = useRef(true);
  useEffect(() => {
    // Only when the answer changes to "in", not when the screen opens.
    if (first.current) {
      first.current = false;
      return;
    }
    if (!jig || reduceMotion) return;
    tilt.set(
      withSequence(
        withTiming(-12, { duration: 140 }),
        withTiming(8, { duration: 160 }),
        withTiming(-4, { duration: 160 }),
        withSpring(0, motion.press),
      ),
    );
    lift.set(withSequence(withTiming(-8, { duration: 140 }), withSpring(0, motion.press)));
  }, [jig, reduceMotion, tilt, lift]);
  const style = useAnimatedStyle(() => ({
    transform: [{ translateY: lift.value }, { rotate: `${tilt.value}deg` }],
  }));
  return (
    <Animated.View style={[styles.tile, { backgroundColor: color }, style]} aria-hidden>
      {cover ? (
        <CoverIcon cover={cover} size={26} color={entryInk} />
      ) : (
        <Icon name="sparkles" size={24} color={entryInk} />
      )}
    </Animated.View>
  );
}

type FaceStatus = 'going' | 'waiting' | 'declined';

const statusLabel: Record<FaceStatus, string> = {
  going: 'going',
  waiting: 'no answer yet',
  declined: 'can’t make it',
};

/** A row of faces: people going in full color, the rest faded. */
function Faces({
  title,
  groups,
  me,
  joined,
}: {
  title: string;
  groups: { members: CalendarMember[]; status: FaceStatus }[];
  me: string;
  /** You just said you're in: your face bounces in. */
  joined: boolean;
}) {
  const styles = useStyles();
  return (
    <View style={styles.faces}>
      <Text variant="caption" tone="secondary" role="heading" aria-level={2} style={styles.facesTitle}>
        {title}
      </Text>
      <View style={styles.faceRow} role="list">
        {groups.flatMap(({ members, status }) =>
          members.map((m) => (
            <Face
              key={m.user.id}
              member={m}
              isMe={m.user.id === me}
              status={status}
              bounce={joined && m.user.id === me}
            />
          )),
        )}
      </View>
    </View>
  );
}

function Face({
  member,
  isMe,
  status,
  bounce,
}: {
  member: CalendarMember;
  isMe: boolean;
  status: FaceStatus;
  bounce: boolean;
}) {
  const styles = useStyles();
  const reduceMotion = useReducedMotion();
  const going = status === 'going';
  const pop = useSharedValue(bounce && !reduceMotion ? 0 : 1);
  useEffect(() => {
    if (!bounce || reduceMotion) return;
    pop.set(0);
    pop.set(withSpring(1, { damping: 9, stiffness: 260, mass: 0.7 }));
  }, [bounce, reduceMotion, pop]);
  const style = useAnimatedStyle(() => ({
    opacity: Math.min(1, 0.2 + pop.value),
    transform: [{ translateY: (1 - pop.value) * 14 }, { scale: 0.6 + 0.4 * pop.value }],
  }));
  const name = isMe ? 'You' : member.user.username;

  return (
    <Animated.View
      style={[styles.face, style]}
      role="listitem"
      aria-label={`@${member.user.username}${isMe ? ' (you)' : ''}, ${statusLabel[status]}`}
    >
      <View style={[styles.faceRing, !going && styles.faceRingWaiting]}>
        <View style={!going && styles.faded}>
          <Avatar id={member.user.id} name={member.user.username} url={member.user.avatarUrl} size={FACE} />
        </View>
      </View>
      <Text variant="caption" tone={going ? 'primary' : 'tertiary'} numberOfLines={1} style={styles.faceName}>
        {name}
      </Text>
    </Animated.View>
  );
}

const COVER_HEIGHT = 260;
/** How far the sheet with the details rides up over the cover. */
const SHEET_OVERLAP = 24;
const FACE = 48;
const TILE = 52;

const useStyles = createStyles((c) => ({
  root: {
    flex: 1,
    backgroundColor: c.background,
  },
  scroll: {
    paddingBottom: space.xxxl,
  },
  inner: {
    width: '100%',
    maxWidth: layout.contentMaxWidth,
    alignSelf: 'center',
  },
  cover: {
    width: '100%',
  },
  coverWide: {
    borderRadius: radius.xl,
  },
  coverText: {
    position: 'absolute',
    left: space.lg + space.xxs,
    right: space.lg + space.xxs,
    // Clear of the sheet that overlaps the cover's bottom.
    bottom: SHEET_OVERLAP + space.md,
    alignItems: 'flex-start',
    gap: space.sm,
  },
  chip: {
    backgroundColor: c.sunset[2],
    borderRadius: radius.pill,
    paddingHorizontal: space.sm + 1,
    paddingVertical: 3,
  },
  chipText: {
    fontWeight: '800',
  },
  title: {
    color: '#FFFFFF',
  },
  sheet: {
    marginTop: -SHEET_OVERLAP,
    backgroundColor: c.background,
    borderTopLeftRadius: radius.xl,
    borderTopRightRadius: radius.xl,
    paddingHorizontal: space.lg,
    paddingTop: space.lg,
  },
  sheetWide: {
    marginHorizontal: space.lg,
    paddingHorizontal: space.xl,
  },
  countdown: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    marginTop: space.xs,
  },
  countdownText: {
    flex: 1,
    gap: 1,
  },
  tile: {
    width: TILE,
    height: TILE,
    borderRadius: radius.md,
    alignItems: 'center',
    justifyContent: 'center',
  },
  answers: {
    flexDirection: 'row',
    gap: space.sm + 2,
    marginTop: space.xl,
  },
  answerWide: {
    flex: 1.4,
  },
  answer: {
    flex: 1,
  },
  answerHint: {
    marginTop: space.sm,
    paddingHorizontal: space.xs,
  },
  faces: {
    marginTop: space.xl,
  },
  facesTitle: {
    paddingHorizontal: space.xs,
    marginBottom: space.md,
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  faceRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: space.md,
  },
  face: {
    width: FACE + space.lg,
    alignItems: 'center',
    gap: space.xs,
  },
  faceRing: {
    padding: 2,
    borderRadius: radius.pill,
    borderWidth: 1.5,
    borderColor: 'transparent',
  },
  faceRingWaiting: {
    borderColor: c.textTertiary,
    borderStyle: 'dashed',
  },
  faded: {
    opacity: 0.4,
  },
  faceName: {
    maxWidth: FACE + space.lg,
    textAlign: 'center',
  },
  floating: {
    position: 'absolute',
    left: 0,
    right: 0,
  },
  floatingRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: space.lg,
  },
  floatingRowWide: {
    width: '100%',
    maxWidth: layout.contentMaxWidth + space.xxl * 2,
    alignSelf: 'center',
    paddingHorizontal: space.xxl + space.md,
  },
  floatingButton: {
    borderRadius: radius.pill,
    backgroundColor: c.background,
    shadowColor: '#000000',
    shadowOpacity: 0.2,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 3 },
    elevation: 3,
  },
}));
