import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { api, ApiError, unwrap, type Calendar, type MemberRole } from './client';

const calendarsQueryKey = ['calendars'] as const;
const calendarQueryKey = (id: string) => [...calendarsQueryKey, id] as const;

export const calendarNameMaxLength = 50;

/** The user's calendars, for the calendars screen. */
export function useCalendars() {
  return useQuery({
    queryKey: calendarsQueryKey,
    queryFn: async () => (await unwrap(api.GET('/calendars'))).calendars,
  });
}

/** True when the calendar is gone or the user is no longer a member. */
export function isCalendarGone(error: unknown): boolean {
  return error instanceof ApiError && error.status === 404;
}

export function useCalendar(id: string) {
  return useQuery({
    queryKey: calendarQueryKey(id),
    queryFn: () => unwrap(api.GET('/calendars/{calendarId}', { params: { path: { calendarId: id } } })),
    retry: (failures, error) => !isCalendarGone(error) && failures < 1,
  });
}

/**
 * Mutations that answer with the updated calendar put it straight into the
 * cache; the list is refetched because names, roles and members show there too.
 */
function useCalendarMutation<V>(id: string, mutationFn: (variables: V) => Promise<Calendar>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: (calendar) => queryClient.setQueryData(calendarQueryKey(id), calendar),
    onError: (error) => {
      if (isCalendarGone(error)) queryClient.invalidateQueries({ queryKey: calendarQueryKey(id) });
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: calendarsQueryKey, exact: true }),
  });
}

/** Mutations after which the user no longer sees the calendar: delete and leave. */
function useCalendarGoneMutation<V>(id: string, mutationFn: (variables: V) => Promise<unknown>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: calendarQueryKey(id) });
      return queryClient.invalidateQueries({ queryKey: calendarsQueryKey, exact: true });
    },
  });
}

export function useCreateCalendar() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => unwrap(api.POST('/calendars', { body: { name } })),
    onSuccess: (calendar) => {
      queryClient.setQueryData(calendarQueryKey(calendar.id), calendar);
      return queryClient.invalidateQueries({ queryKey: calendarsQueryKey, exact: true });
    },
  });
}

export function useUpdateCalendar(id: string) {
  return useCalendarMutation(id, (body: { name?: string; membersCanInvite?: boolean }) =>
    unwrap(api.PATCH('/calendars/{calendarId}', { params: { path: { calendarId: id } }, body })),
  );
}

export function useDeleteCalendar(id: string) {
  return useCalendarGoneMutation(id, () =>
    unwrap(api.DELETE('/calendars/{calendarId}', { params: { path: { calendarId: id } } })),
  );
}

export function useInviteMember(id: string) {
  return useCalendarMutation(id, (body: { userId: string; role?: MemberRole }) =>
    unwrap(api.POST('/calendars/{calendarId}/members', { params: { path: { calendarId: id } }, body })),
  );
}

export function useUpdateMember(id: string) {
  return useCalendarMutation(id, ({ userId, ...body }: { userId: string; role?: MemberRole; canInvite?: boolean }) =>
    unwrap(
      api.PATCH('/calendars/{calendarId}/members/{userId}', { params: { path: { calendarId: id, userId } }, body }),
    ),
  );
}

/** Removes another member; the calendar is refetched to drop them from the list. */
export function useRemoveMember(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      unwrap(api.DELETE('/calendars/{calendarId}/members/{userId}', { params: { path: { calendarId: id, userId } } })),
    onSettled: () => queryClient.invalidateQueries({ queryKey: calendarsQueryKey }),
  });
}

/** Leaving is removing yourself. */
export function useLeaveCalendar(id: string) {
  return useCalendarGoneMutation(id, (userId: string) =>
    unwrap(api.DELETE('/calendars/{calendarId}/members/{userId}', { params: { path: { calendarId: id, userId } } })),
  );
}

export function useTransferOwnership(id: string) {
  return useCalendarMutation(id, (userId: string) =>
    unwrap(api.POST('/calendars/{calendarId}/transfer', { params: { path: { calendarId: id } }, body: { userId } })),
  );
}

export const roleLabel: Record<Calendar['role'], string> = {
  owner: 'Owner',
  editor: 'Editor',
  viewer: 'Viewer',
};

export const roleDescription: Record<MemberRole, string> = {
  editor: 'Can add and edit entries.',
  viewer: 'Can see entries and accept or decline them.',
};
