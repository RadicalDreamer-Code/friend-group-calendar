import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, View } from 'react-native';

import { canEditEntries, useCalendar } from '@/api/calendars';
import { errorMessage, type Entry, type EntryColor, type EntryCover } from '@/api/client';
import {
  entryMaxDays,
  entryTitleMaxLength,
  isEntryGone,
  useCreateEntry,
  useDeleteEntry,
  useEntry,
  useUpdateEntry,
  type EntryFields,
} from '@/api/entries';
import { Button } from '@/components/Button';
import { CalendarError } from '@/components/CalendarError';
import { Cover } from '@/components/Cover';
import { EmptyState } from '@/components/EmptyState';
import { Icon } from '@/components/Icon';
import { ListRow, ListSection } from '@/components/ListSection';
import { MonthCalendar } from '@/components/MonthCalendar';
import { Page } from '@/components/Page';
import { SkeletonList } from '@/components/Skeleton';
import { Switch } from '@/components/Switch';
import { Text } from '@/components/Text';
import { TextField } from '@/components/TextField';
import { confirm } from '@/lib/confirm';
import { coverLabels, coverScenes } from '@/lib/covers';
import { addDays, daysFrom, formatDay, formatShortDay, isISODate, monthOf, today } from '@/lib/dates';
import { haptics } from '@/lib/haptics';
import { useEscapeKey } from '@/lib/useEscapeKey';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { entryColor, entryColors, entryInk, radius, space } from '@/theme/tokens';

export { ScreenError as ErrorBoundary } from '@/components/ScreenError';

const colorNames: Record<EntryColor, string> = {
  fjord: 'Lake blue',
  sage: 'Sage green',
  clay: 'Burnt orange',
  rose: 'Salmon pink',
  plum: 'Plum',
  sand: 'Mustard',
  moss: 'Avocado',
  slate: 'Taupe',
};

const close = () => (router.canGoBack() ? router.back() : router.replace('/'));

/**
 * The sheet for adding an entry (with an optional `date` to start on) or,
 * with `entryId`, editing and deleting one. Owner and editors only.
 */
export default function EditEntryScreen() {
  const { calendarId, entryId, date } = useLocalSearchParams<{ calendarId: string; entryId?: string; date?: string }>();
  const title = entryId ? 'Edit entry' : 'New entry';
  const calendar = useCalendar(calendarId);
  useEscapeKey(close);

  if (calendar.isPending) {
    return (
      <Page title={title} sheet>
        <SkeletonList rows={3} />
      </Page>
    );
  }
  if (calendar.data === undefined) {
    return (
      <Page title={title} sheet>
        <CalendarError error={calendar.error} onRetry={calendar.refetch} retrying={calendar.isFetching} />
      </Page>
    );
  }
  if (!canEditEntries(calendar.data.role)) {
    return (
      <Page title={title} sheet>
        <EmptyState
          icon="lock-closed-outline"
          title="Only editors can do this"
          message="Ask the owner to make you an editor to add and change entries."
          action={<Button variant="secondary" title="Close" onPress={close} />}
        />
      </Page>
    );
  }
  if (entryId) return <EditExisting calendarId={calendarId} entryId={entryId} />;
  const start = isISODate(date) ? date : today();
  return (
    <Page title={title} sheet>
      <EntryForm
        calendarId={calendarId}
        initial={{ title: '', date: start, endDate: start, color: 'fjord', cover: null }}
      />
    </Page>
  );
}

function EditExisting({ calendarId, entryId }: { calendarId: string; entryId: string }) {
  const entry = useEntry(calendarId, entryId);
  if (entry.isPending) {
    return (
      <Page title="Edit entry" sheet>
        <SkeletonList rows={3} />
      </Page>
    );
  }
  if (entry.isError) {
    return (
      <Page title="Edit entry" sheet>
        {isEntryGone(entry.error) ? (
          <EmptyState
            icon="calendar-clear-outline"
            title="Entry deleted"
            message="Someone removed it from the calendar."
            action={<Button variant="secondary" title="Close" onPress={close} />}
          />
        ) : (
          <CalendarError error={entry.error} onRetry={entry.refetch} retrying={entry.isFetching} />
        )}
      </Page>
    );
  }
  return (
    <Page title="Edit entry" sheet>
      <EntryForm calendarId={calendarId} entry={entry.data} initial={entry.data} />
    </Page>
  );
}

function EntryForm({ calendarId, entry, initial }: { calendarId: string; entry?: Entry; initial: EntryFields }) {
  const styles = useStyles();
  // What the form opened with; `initial` follows the polled entry.
  const [original] = useState(initial);
  const [title, setTitle] = useState(initial.title);
  const [date, setDate] = useState(initial.date);
  const [endDate, setEndDate] = useState(initial.endDate);
  // "Ends on" is off for a one-day entry. While it is on, the first tap on the
  // grid picks the first day and the next one the last day.
  const [multiDay, setMultiDay] = useState(initial.endDate !== initial.date);
  const [pickingEnd, setPickingEnd] = useState(false);
  const [month, setMonth] = useState(monthOf(initial.date));
  const [color, setColor] = useState<EntryColor>(initial.color);
  const [cover, setCover] = useState<EntryCover | null>(initial.cover);
  const create = useCreateEntry(calendarId);
  const update = useUpdateEntry(calendarId, entry?.id ?? '');
  const save = entry ? update : create;
  const trimmed = title.trim();
  const days = daysFrom(endDate, date) + 1;
  const moved =
    entry &&
    (date !== original.date || endDate !== original.endDate) &&
    (entry.date !== date || entry.endDate !== endDate) &&
    entry.accepted.length + entry.declined.length > 0;

  const pickDay = (d: string) => {
    setMonth(monthOf(d));
    if (multiDay && pickingEnd && d >= date && daysFrom(d, date) < entryMaxDays) {
      setEndDate(d);
      setPickingEnd(false);
      return;
    }
    // A new first day; with "Ends on", the next tap picks the last one.
    setDate(d);
    setEndDate(d);
    setPickingEnd(multiDay);
  };

  const toggleMultiDay = (on: boolean) => {
    setMultiDay(on);
    setPickingEnd(on);
    if (!on) setEndDate(date);
  };

  const submit = () => {
    if (trimmed === '' || save.isPending) return;
    const body = { title: trimmed, date, endDate, color, cover };
    // Only what the user changed, so edits someone else made meanwhile stay.
    // A new first day goes with its last day: on its own the API keeps the length.
    const changes: Partial<EntryFields> = {};
    if (trimmed !== original.title) changes.title = trimmed;
    if (date !== original.date) changes.date = date;
    if (date !== original.date || endDate !== original.endDate) changes.endDate = endDate;
    if (color !== original.color) changes.color = color;
    if (cover !== original.cover) changes.cover = cover;
    const options = {
      // Back to the calendar on the entry's day, or to the entry itself.
      onSuccess: (saved: Entry) => {
        haptics.success();
        if (entry) close();
        else router.dismissTo({ pathname: '/calendars/[id]', params: { id: calendarId, day: saved.date } });
      },
    };
    if (entry) update.mutate(changes, options);
    else create.mutate(body, options);
  };

  return (
    <>
      <View style={styles.form}>
        <TextField
          label="Name"
          value={title}
          onChangeText={(text) => {
            setTitle(text);
            if (save.isError) save.reset();
          }}
          onSubmitEditing={submit}
          placeholder="e.g. Board game night"
          autoFocus={!entry}
          autoCapitalize="sentences"
          maxLength={entryTitleMaxLength}
          returnKeyType="done"
        />

        <View>
          <View style={styles.dateHeader}>
            <View style={styles.dateLabel}>
              <Text variant="footnote" tone="secondary">
                {multiDay ? 'Dates' : 'Date'}
              </Text>
              <Text variant="subhead" aria-live="polite">
                {days > 1 ? `${formatShortDay(date)} → ${formatShortDay(endDate)} · ${days} days` : formatDay(date)}
              </Text>
            </View>
            <View style={styles.endsOn}>
              <Text variant="footnote" tone="secondary">
                Ends on
              </Text>
              <Switch label="Ends on another day" value={multiDay} onValueChange={toggleMultiDay} />
            </View>
          </View>
          <View style={styles.datePicker}>
            <MonthCalendar
              compact
              month={month}
              onMonthChange={setMonth}
              selected={date}
              rangeEnd={multiDay ? endDate : null}
              onSelect={pickDay}
            />
          </View>
          {multiDay && pickingEnd ? (
            <Text variant="footnote" tone="tertiary" style={styles.note}>
              Now tap the last day, up to {formatShortDay(addDays(date, entryMaxDays - 1))}.
            </Text>
          ) : null}
          {moved ? (
            <Text variant="footnote" tone="tertiary" style={styles.note}>
              Changing the dates clears everyone’s answers.
            </Text>
          ) : null}
        </View>

        <View>
          <Text variant="footnote" tone="secondary" style={styles.fieldLabel}>
            Color
          </Text>
          <ColorPicker value={color} onChange={setColor} />
        </View>

        <View>
          <Text variant="footnote" tone="secondary" style={styles.fieldLabel}>
            Cover
          </Text>
          <CoverPicker value={cover} color={entryColor(color)} onChange={setCover} />
        </View>

        {save.isError ? (
          <Text variant="footnote" tone="danger">
            {errorMessage(save.error)}
          </Text>
        ) : null}
        <Button
          title={entry ? 'Save' : 'Add entry'}
          onPress={submit}
          loading={save.isPending}
          disabled={trimmed === ''}
        />
      </View>
      <View style={styles.cancel}>
        <Button variant="plain" title="Cancel" onPress={close} />
      </View>
      {entry ? <DeleteEntry calendarId={calendarId} entry={entry} /> : null}
    </>
  );
}

function ColorPicker({ value, onChange }: { value: EntryColor; onChange: (color: EntryColor) => void }) {
  const styles = useStyles();
  const { colors } = useTheme();
  return (
    <View style={styles.colors} role="radiogroup" aria-label="Color">
      {entryColors.map((c) => {
        const selected = c.id === value;
        return (
          <Pressable
            key={c.id}
            role="radio"
            aria-checked={selected}
            aria-label={colorNames[c.id]}
            onPress={() => {
              if (c.id !== value) haptics.selection();
              onChange(c.id);
            }}
            hitSlop={4}
            style={[styles.colorRing, selected && { borderColor: colors.text }]}
          >
            <View style={[styles.color, { backgroundColor: c.value }]}>
              {selected ? <Icon name="checkmark" size={18} color={entryInk} /> : null}
            </View>
          </Pressable>
        );
      })}
    </View>
  );
}

/** "None" (the sunset in the entry's color) and every scene, four to a row. */
function CoverPicker({
  value,
  color,
  onChange,
}: {
  value: EntryCover | null;
  color: string;
  onChange: (cover: EntryCover | null) => void;
}) {
  const styles = useStyles();
  const { colors } = useTheme();
  const options = [null, ...coverScenes];
  return (
    <View style={styles.covers} role="radiogroup" aria-label="Cover">
      {options.map((c) => {
        const selected = c === value;
        const label = c ? coverLabels[c] : 'None';
        return (
          <Pressable
            key={c ?? 'none'}
            role="radio"
            aria-checked={selected}
            aria-label={label}
            onPress={() => {
              if (c !== value) haptics.selection();
              onChange(c);
            }}
            style={styles.coverOption}
          >
            <View style={[styles.coverRing, selected && { borderColor: colors.text }]}>
              <Cover color={color} cover={c} style={styles.coverThumb}>
                {selected ? (
                  <View style={styles.coverCheck}>
                    <Icon name="checkmark" size={14} color={entryInk} />
                  </View>
                ) : null}
              </Cover>
            </View>
            <Text
              variant="caption"
              tone={selected ? 'primary' : 'secondary'}
              numberOfLines={1}
              style={styles.coverLabel}
            >
              {label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

function DeleteEntry({ calendarId, entry }: { calendarId: string; entry: Entry }) {
  const remove = useDeleteEntry(calendarId, entry.id);
  const onDelete = async () => {
    const ok = await confirm({
      title: `Delete “${entry.title}”?`,
      message: 'It disappears for everyone in the calendar, with all answers.',
      confirmLabel: 'Delete',
      destructive: true,
    });
    if (ok) {
      remove.mutate(undefined, {
        onSuccess: () => router.dismissTo({ pathname: '/calendars/[id]', params: { id: calendarId, day: entry.date } }),
      });
    }
  };
  return (
    <ListSection footer={remove.isError ? errorMessage(remove.error) : undefined}>
      <ListRow label="Delete entry" action="destructive" onPress={onDelete} disabled={remove.isPending} />
    </ListSection>
  );
}

const SWATCH = 32;
const RING = 2;

const useStyles = createStyles(() => ({
  form: {
    gap: space.xl,
    marginTop: space.md,
  },
  dateHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
    paddingHorizontal: space.lg,
    marginBottom: space.sm,
  },
  dateLabel: {
    flex: 1,
    gap: space.xxs,
  },
  endsOn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.sm,
  },
  fieldLabel: {
    paddingHorizontal: space.lg,
    marginBottom: space.sm,
  },
  datePicker: {
    paddingHorizontal: space.xs,
  },
  note: {
    paddingHorizontal: space.lg,
    marginTop: space.sm,
  },
  // All eight fit in one row on a 375 pt phone.
  colors: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: space.xs,
  },
  colorRing: {
    padding: RING,
    borderRadius: radius.pill,
    borderWidth: RING,
    borderColor: 'transparent',
  },
  color: {
    width: SWATCH,
    height: SWATCH,
    borderRadius: radius.pill,
    alignItems: 'center',
    justifyContent: 'center',
  },
  // None and the seven scenes: two rows of four.
  covers: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    rowGap: space.md,
    paddingHorizontal: space.xs,
  },
  coverOption: {
    width: '25%',
    paddingHorizontal: space.xxs,
    alignItems: 'center',
    gap: space.xs,
  },
  coverRing: {
    alignSelf: 'stretch',
    padding: RING,
    borderRadius: radius.sm + RING * 2,
    borderWidth: RING,
    borderColor: 'transparent',
  },
  coverThumb: {
    aspectRatio: 5 / 3,
    borderRadius: radius.sm,
  },
  coverCheck: {
    position: 'absolute',
    top: space.xs,
    right: space.xs,
    width: 20,
    height: 20,
    borderRadius: radius.pill,
    backgroundColor: '#FFF8EE',
    alignItems: 'center',
    justifyContent: 'center',
  },
  coverLabel: {
    textAlign: 'center',
  },
  cancel: {
    marginTop: space.sm,
    alignItems: 'center',
  },
}));
