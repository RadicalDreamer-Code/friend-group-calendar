import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query';

import { monthOf } from '@/lib/dates';
import { usePollWhileFocused } from '@/lib/freshness';

import { calendarQueryKey, isCalendarGone } from './calendars';
import { api, ApiError, unwrap, type Entry, type EntryColor, type EntryResponseStatus } from './client';

export const entryTitleMaxLength = 80;

const entriesQueryKey = (calendarId: string) => [...calendarQueryKey(calendarId), 'entries'] as const;
const monthQueryKey = (calendarId: string, month: string) => [...entriesQueryKey(calendarId), month] as const;
const entryQueryKey = (calendarId: string, entryId: string) =>
  [...calendarQueryKey(calendarId), 'entry', entryId] as const;

/** A member's answer; null means they have not answered yet. */
export type Reply = EntryResponseStatus | null;

export function replyOf(entry: Entry, userId: string): Reply {
  if (entry.accepted.includes(userId)) return 'accepted';
  if (entry.declined.includes(userId)) return 'declined';
  return null;
}

/** The entry with the user's answer changed, like the API will answer. */
export function withReply(entry: Entry, userId: string, reply: Reply): Entry {
  const others = (ids: string[]) => ids.filter((id) => id !== userId);
  return {
    ...entry,
    accepted: reply === 'accepted' ? [...others(entry.accepted), userId] : others(entry.accepted),
    declined: reply === 'declined' ? [...others(entry.declined), userId] : others(entry.declined),
  };
}

/** True when the entry was deleted. */
export function isEntryGone(error: unknown): boolean {
  return error instanceof ApiError && error.code === 'entry_not_found';
}

/** One month of a calendar's entries, polled while the screen is focused. */
export function useMonthEntries(calendarId: string, month: string) {
  return useQuery({
    queryKey: monthQueryKey(calendarId, month),
    queryFn: async () =>
      (
        await unwrap(
          api.GET('/calendars/{calendarId}/entries', {
            params: { path: { calendarId }, query: { month } },
          }),
        )
      ).entries,
    refetchInterval: usePollWhileFocused(),
    retry: (failures, error) => !isCalendarGone(error) && failures < 1,
  });
}

/** One entry, shown straight away from a loaded month when there is one. */
export function useEntry(calendarId: string, entryId: string) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: entryQueryKey(calendarId, entryId),
    queryFn: () =>
      unwrap(
        api.GET('/calendars/{calendarId}/entries/{entryId}', {
          params: { path: { calendarId, entryId } },
        }),
      ),
    initialData: () => findCached(queryClient, calendarId, entryId)?.entry,
    initialDataUpdatedAt: () => findCached(queryClient, calendarId, entryId)?.updatedAt,
    refetchInterval: usePollWhileFocused(),
    retry: (failures, error) => !isCalendarGone(error) && !isEntryGone(error) && failures < 1,
  });
}

function findCached(queryClient: QueryClient, calendarId: string, entryId: string) {
  for (const query of queryClient.getQueryCache().findAll({ queryKey: entriesQueryKey(calendarId) })) {
    const entry = (query.state.data as Entry[] | undefined)?.find((e) => e.id === entryId);
    if (entry) return { entry, updatedAt: query.state.dataUpdatedAt };
  }
  return undefined;
}

const byDay = (a: Entry, b: Entry) =>
  a.date.localeCompare(b.date) || a.createdAt.localeCompare(b.createdAt) || a.id.localeCompare(b.id);

/**
 * Puts an entry everywhere it is shown: its own query and the month it is
 * in. It leaves any other month, which matters when its date moved.
 */
function putEntry(queryClient: QueryClient, calendarId: string, entry: Entry) {
  queryClient.setQueryData(entryQueryKey(calendarId, entry.id), entry);
  for (const [key, list] of queryClient.getQueriesData<Entry[]>({ queryKey: entriesQueryKey(calendarId) })) {
    if (!list) continue;
    const month = key[key.length - 1];
    const others = list.filter((e) => e.id !== entry.id);
    queryClient.setQueryData(key, month === monthOf(entry.date) ? [...others, entry].sort(byDay) : others);
  }
}

function dropEntry(queryClient: QueryClient, calendarId: string, entryId: string) {
  queryClient.removeQueries({ queryKey: entryQueryKey(calendarId, entryId) });
  for (const [key, list] of queryClient.getQueriesData<Entry[]>({ queryKey: entriesQueryKey(calendarId) })) {
    if (list)
      queryClient.setQueryData(
        key,
        list.filter((e) => e.id !== entryId),
      );
  }
}

/** After a failed change the cached entries may be wrong, so they are refetched. */
function refetchAfterError(queryClient: QueryClient, calendarId: string, error: unknown) {
  // A calendar that is gone also drops its entries, which share the key prefix.
  return queryClient.invalidateQueries({
    queryKey: isCalendarGone(error) ? calendarQueryKey(calendarId) : entriesQueryKey(calendarId),
  });
}

export type EntryFields = { title: string; date: string; color: EntryColor };

export function useCreateEntry(calendarId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: EntryFields) =>
      unwrap(api.POST('/calendars/{calendarId}/entries', { params: { path: { calendarId } }, body })),
    onSuccess: (entry) => putEntry(queryClient, calendarId, entry),
    onError: (error) => refetchAfterError(queryClient, calendarId, error),
  });
}

export function useUpdateEntry(calendarId: string, entryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<EntryFields>) =>
      unwrap(
        api.PATCH('/calendars/{calendarId}/entries/{entryId}', {
          params: { path: { calendarId, entryId } },
          body,
        }),
      ),
    scope: { id: `entry:${entryId}` },
    onSuccess: (entry) => putEntry(queryClient, calendarId, entry),
    onError: (error) => {
      if (isEntryGone(error)) dropEntry(queryClient, calendarId, entryId);
      return refetchAfterError(queryClient, calendarId, error);
    },
  });
}

export function useDeleteEntry(calendarId: string, entryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE('/calendars/{calendarId}/entries/{entryId}', {
          params: { path: { calendarId, entryId } },
        }),
      ),
    scope: { id: `entry:${entryId}` },
    onSuccess: () => dropEntry(queryClient, calendarId, entryId),
    onError: (error) => {
      // Someone else deleted it first: that is what the user wanted.
      if (isEntryGone(error)) dropEntry(queryClient, calendarId, entryId);
      else return refetchAfterError(queryClient, calendarId, error);
    },
  });
}

/**
 * Accepts, declines or (with null) takes the answer back. The answer shows
 * right away; answers to one entry are sent one after another, so the last
 * tap wins.
 */
export function useRespond(calendarId: string, entryId: string, userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reply: Reply) => {
      const params = { path: { calendarId, entryId } };
      return unwrap(
        reply
          ? api.PUT('/calendars/{calendarId}/entries/{entryId}/response', { params, body: { status: reply } })
          : api.DELETE('/calendars/{calendarId}/entries/{entryId}/response', { params }),
      );
    },
    scope: { id: `entry:${entryId}` },
    onMutate: async (reply) => {
      // Polls in flight must not overwrite the answer with what it was before.
      await queryClient.cancelQueries({ queryKey: entriesQueryKey(calendarId) });
      await queryClient.cancelQueries({ queryKey: entryQueryKey(calendarId, entryId) });
      const entry =
        queryClient.getQueryData<Entry>(entryQueryKey(calendarId, entryId)) ??
        findCached(queryClient, calendarId, entryId)?.entry;
      if (entry) putEntry(queryClient, calendarId, withReply(entry, userId, reply));
    },
    onSuccess: (entry) => {
      // Only the newest answer's response is the truth; older ones would flicker.
      const inFlight = queryClient.isMutating({ predicate: (m) => m.options.scope?.id === `entry:${entryId}` });
      if (inFlight <= 1) putEntry(queryClient, calendarId, entry);
    },
    onError: (error) => {
      if (isEntryGone(error)) dropEntry(queryClient, calendarId, entryId);
      return refetchAfterError(queryClient, calendarId, error);
    },
  });
}
