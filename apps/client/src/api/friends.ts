import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';

import { api, unwrap } from './client';

const friendsQueryKey = ['friends'] as const;
const searchQueryKey = ['user-search'] as const;

/** Friends plus incoming and outgoing requests, for the friends screen and the tab badge. */
export function useFriends() {
  return useQuery({
    queryKey: friendsQueryKey,
    queryFn: () => unwrap(api.GET('/friends')),
  });
}

/** Number of requests waiting for the user to answer. */
export function useIncomingRequestCount(): number {
  return useFriends().data?.incoming.length ?? 0;
}

/** What someone typed as a search: "@Ada " → "ada". */
export function normalizeSearch(text: string): string {
  return text.trim().replace(/^@/, '').toLowerCase();
}

export const searchPattern = /^[a-z0-9_.]{2,20}$/;

/** Searches usernames as the user types, after a short pause. */
export function useUserSearch(text: string) {
  const query = normalizeSearch(text);
  const [debounced, setDebounced] = useState(query);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(id);
  }, [query]);

  const search = useQuery({
    queryKey: [...searchQueryKey, debounced],
    queryFn: async () => (await unwrap(api.GET('/users/search', { params: { query: { q: debounced } } }))).results,
    enabled: searchPattern.test(debounced),
    staleTime: 10_000,
    // Keep the last results on screen while the next ones load.
    placeholderData: keepPreviousData,
  });
  return { ...search, query, settled: query === debounced };
}

/** Every friends mutation changes both the lists and the statuses in search results. */
function useFriendsMutation<T>(mutationFn: (userId: string) => Promise<T>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: friendsQueryKey }),
        queryClient.invalidateQueries({ queryKey: searchQueryKey }),
      ]),
  });
}

/** Resolves to "friends" when they had already asked, which accepts their request. */
export function useSendFriendRequest() {
  return useFriendsMutation(
    async (userId) => (await unwrap(api.POST('/friend-requests', { body: { userId } }))).friendship,
  );
}

export function useAcceptFriendRequest() {
  return useFriendsMutation((userId) =>
    unwrap(api.POST('/friend-requests/{userId}/accept', { params: { path: { userId } } })),
  );
}

/** Declines a request from this user, or cancels one sent to them. */
export function useDeleteFriendRequest() {
  return useFriendsMutation((userId) =>
    unwrap(api.DELETE('/friend-requests/{userId}', { params: { path: { userId } } })),
  );
}

export function useRemoveFriend() {
  return useFriendsMutation((userId) => unwrap(api.DELETE('/friends/{userId}', { params: { path: { userId } } })));
}
