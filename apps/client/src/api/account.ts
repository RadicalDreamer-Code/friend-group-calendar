import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import * as Device from 'expo-device';
import type { ImagePickerAsset } from 'expo-image-picker';
import { useEffect, useState } from 'react';
import { Platform } from 'react-native';

import { clearToken, saveToken } from '@/auth/token';
import { meQueryKey } from '@/auth/session';

import { api, unwrap, type Platform as ApiPlatform, type User } from './client';

const sessionsQueryKey = ['sessions'] as const;

/** Resolves to the challenge ID that verifying this code needs. */
export function useRequestCode() {
  return useMutation({
    mutationFn: async (email: string) => {
      const res = await unwrap(api.POST('/auth/request-code', { body: { email } }));
      return res.challengeId;
    },
  });
}

export function useVerifyCode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ challengeId, email, code }: { challengeId: string; email: string; code: string }) => {
      const platform = Platform.OS as ApiPlatform;
      const res = await unwrap(
        api.POST('/auth/verify', {
          body: {
            challengeId,
            email,
            code,
            platform,
            // Web: the API names the browser from its user agent.
            deviceName: platform === 'web' ? undefined : (Device.modelName ?? undefined),
          },
        }),
      );
      if (res.token) await saveToken(res.token);
      return res.user;
    },
    onSuccess: (user) => {
      // A session that ended with a 401 (revoked or expired) leaves the
      // previous user's friends and searches cached; never show them to this one.
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== meQueryKey[0] });
      queryClient.setQueryData(meQueryKey, user);
    },
  });
}

export function useSignOut() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      // Sign out locally even if the server cannot be reached.
      await api.POST('/auth/logout').catch(() => {});
      await clearToken();
    },
    onSettled: () => {
      queryClient.clear();
      queryClient.setQueryData(meQueryKey, null);
    },
  });
}

function useSetMe() {
  const queryClient = useQueryClient();
  return (user: User) => queryClient.setQueryData(meQueryKey, user);
}

export function useUpdateUsername() {
  const setMe = useSetMe();
  return useMutation({
    mutationFn: (username: string) => unwrap(api.PATCH('/me', { body: { username } })),
    onSuccess: setMe,
  });
}

export const usernamePattern = /^[a-z0-9_.]{3,20}$/;

export type UsernameCheck = 'idle' | 'invalid' | 'checking' | 'available' | 'taken' | 'unknown';

/** Checks a username as the user types, after a short pause. */
export function useUsernameCheck(username: string): UsernameCheck {
  const [debounced, setDebounced] = useState(username);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(username), 350);
    return () => clearTimeout(id);
  }, [username]);

  const valid = usernamePattern.test(debounced);
  const query = useQuery({
    queryKey: ['username-availability', debounced],
    queryFn: () => unwrap(api.GET('/username-availability', { params: { query: { username: debounced } } })),
    enabled: valid,
    staleTime: 10_000,
    retry: false,
  });

  if (username === '') return 'idle';
  if (!usernamePattern.test(username)) return 'invalid';
  if (username !== debounced || query.isPending) return 'checking';
  if (query.isError) return 'unknown';
  return query.data.available ? 'available' : 'taken';
}

export function useUploadAvatar() {
  const setMe = useSetMe();
  return useMutation({
    mutationFn: (asset: ImagePickerAsset) =>
      unwrap(
        api.PUT('/me/avatar', {
          body: { file: '' },
          bodySerializer: () => {
            const form = new FormData();
            if (asset.file) {
              form.append('file', asset.file);
            } else {
              // React Native's FormData uploads a local file from {uri, name, type}.
              form.append('file', {
                uri: asset.uri,
                name: asset.fileName ?? 'avatar.jpg',
                type: asset.mimeType ?? 'image/jpeg',
              } as unknown as Blob);
            }
            return form;
          },
        }),
      ),
    onSuccess: setMe,
  });
}

export function useRemoveAvatar() {
  const setMe = useSetMe();
  return useMutation({
    mutationFn: () => unwrap(api.DELETE('/me/avatar')),
    onSuccess: setMe,
  });
}

export function useSessions() {
  return useQuery({
    queryKey: sessionsQueryKey,
    queryFn: async () => (await unwrap(api.GET('/me/sessions'))).sessions,
  });
}

export function useRevokeSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (sessionId: string) =>
      unwrap(api.DELETE('/me/sessions/{sessionId}', { params: { path: { sessionId } } })),
    onSettled: () => queryClient.invalidateQueries({ queryKey: sessionsQueryKey }),
  });
}

export function useRevokeOtherSessions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => unwrap(api.POST('/me/sessions/revoke-others')),
    onSettled: () => queryClient.invalidateQueries({ queryKey: sessionsQueryKey }),
  });
}
