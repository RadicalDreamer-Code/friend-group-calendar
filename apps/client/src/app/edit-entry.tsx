import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Pressable, View } from 'react-native';

import { canEditEntries, useCalendar } from '@/api/calendars';
import { errorMessage, type Entry, type EntryColor } from '@/api/client';
import {
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
import { EmptyState } from '@/components/EmptyState';
import { Icon } from '@/components/Icon';
import { ListRow, ListSection } from '@/components/ListSection';
import { MonthCalendar } from '@/components/MonthCalendar';
import { Page } from '@/components/Page';
import { Text } from '@/components/Text';
import { TextField } from '@/components/TextField';
import { confirm } from '@/lib/confirm';
import { formatDay, isISODate, monthOf, today } from '@/lib/dates';
import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { entryColors, radius, space } from '@/theme/tokens';

const colorNames: Record<EntryColor, string> = {
  fjord: 'Fjord blue',
  sage: 'Sage green',
  clay: 'Clay orange',
  rose: 'Rose',
  plum: 'Plum',
  sand: 'Sand yellow',
  moss: 'Moss green',
  slate: 'Slate grey',
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

  if (calendar.isPending) {
    return (
      <Page title={title}>
        <ListSection>
          <ListRow label="Loading…" />
        </ListSection>
      </Page>
    );
  }
  if (calendar.isError) {
    return (
      <Page title={title}>
        <CalendarError error={calendar.error} onRetry={() => calendar.refetch()} />
      </Page>
    );
  }
  if (!canEditEntries(calendar.data.role)) {
    return (
      <Page title={title}>
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
  return (
    <Page title={title}>
      <EntryForm
        calendarId={calendarId}
        initial={{ title: '', date: isISODate(date) ? date : today(), color: 'fjord' }}
      />
    </Page>
  );
}

function EditExisting({ calendarId, entryId }: { calendarId: string; entryId: string }) {
  const entry = useEntry(calendarId, entryId);
  if (entry.isPending) {
    return (
      <Page title="Edit entry">
        <ListSection>
          <ListRow label="Loading…" />
        </ListSection>
      </Page>
    );
  }
  if (entry.isError) {
    return (
      <Page title="Edit entry">
        {isEntryGone(entry.error) ? (
          <EmptyState
            icon="calendar-clear-outline"
            title="Entry deleted"
            message="Someone removed it from the calendar."
            action={<Button variant="secondary" title="Close" onPress={close} />}
          />
        ) : (
          <CalendarError error={entry.error} onRetry={() => entry.refetch()} />
        )}
      </Page>
    );
  }
  return (
    <Page title="Edit entry">
      <EntryForm calendarId={calendarId} entry={entry.data} initial={entry.data} />
    </Page>
  );
}

function EntryForm({
  calendarId,
  entry,
  initial,
}: {
  calendarId: string;
  entry?: Entry;
  initial: { title: string; date: string; color: EntryColor };
}) {
  const styles = useStyles();
  // What the form opened with; `initial` follows the polled entry.
  const [original] = useState(initial);
  const [title, setTitle] = useState(initial.title);
  const [date, setDate] = useState(initial.date);
  const [month, setMonth] = useState(monthOf(initial.date));
  const [color, setColor] = useState<EntryColor>(initial.color);
  const create = useCreateEntry(calendarId);
  const update = useUpdateEntry(calendarId, entry?.id ?? '');
  const save = entry ? update : create;
  const trimmed = title.trim();
  const moved =
    entry && date !== original.date && entry.date !== date && entry.accepted.length + entry.declined.length > 0;

  const submit = () => {
    if (trimmed === '' || save.isPending) return;
    const body = { title: trimmed, date, color };
    // Only what the user changed, so edits someone else made meanwhile stay.
    const changes: Partial<EntryFields> = {};
    if (trimmed !== original.title) changes.title = trimmed;
    if (date !== original.date) changes.date = date;
    if (color !== original.color) changes.color = color;
    const options = {
      // Back to the calendar on the entry's day, or to the entry itself.
      onSuccess: (saved: Entry) =>
        entry
          ? close()
          : router.dismissTo({ pathname: '/calendars/[id]', params: { id: calendarId, day: saved.date } }),
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
          <View style={styles.fieldHeader}>
            <Text variant="footnote" tone="secondary">
              Date
            </Text>
            <Text variant="footnote" tone="secondary">
              {formatDay(date)}
            </Text>
          </View>
          <View style={styles.datePicker}>
            <MonthCalendar
              compact
              month={month}
              onMonthChange={setMonth}
              selected={date}
              onSelect={(d) => {
                setDate(d);
                setMonth(monthOf(d));
              }}
            />
          </View>
          {moved ? (
            <Text variant="footnote" tone="tertiary" style={styles.note}>
              Moving it to another day clears everyone’s answers.
            </Text>
          ) : null}
        </View>

        <View>
          <Text variant="footnote" tone="secondary" style={styles.fieldLabel}>
            Color
          </Text>
          <ColorPicker value={color} onChange={setColor} />
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
            onPress={() => onChange(c.id)}
            hitSlop={4}
            style={[styles.colorRing, selected && { borderColor: colors.text }]}
          >
            <View style={[styles.color, { backgroundColor: c.value }]}>
              {selected ? <Icon name="checkmark" size={18} color="#FFFFFF" /> : null}
            </View>
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
  fieldHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: space.lg,
    marginBottom: space.sm,
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
  cancel: {
    marginTop: space.sm,
    alignItems: 'center',
  },
}));
