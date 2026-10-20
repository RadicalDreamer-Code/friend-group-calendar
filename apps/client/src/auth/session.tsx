import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createContext, use, useEffect, useMemo, useState, type ReactNode } from 'react';

import { api, ApiError, setUnauthorizedHandler, unwrap, type User } from '@/api/client';

import { clearToken, loadToken } from './token';

/**
 * - loading: checking for an existing session
 * - signedOut: show the sign-in screens
 * - needsUsername: signed in for the first time; pick a username before anything else
 * - signedIn: the app
 */
export type SessionStatus = 'loading' | 'signedOut' | 'needsUsername' | 'signedIn';

export const meQueryKey = ['me'] as const;

type SessionContextValue = {
  status: SessionStatus;
  /** The signed-in user; null while loading or signed out. */
  user: User | null;
};

const SessionContext = createContext<SessionContextValue | null>(null);

async function fetchMe(): Promise<User | null> {
  try {
    return await unwrap(api.GET('/me'));
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return null;
    throw e;
  }
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  // Native reads the token from secure storage first; web has nothing to load.
  const [tokenLoaded, setTokenLoaded] = useState(false);
  useEffect(() => {
    loadToken().then(() => setTokenLoaded(true));
  }, []);

  const me = useQuery({
    queryKey: meQueryKey,
    queryFn: fetchMe,
    enabled: tokenLoaded,
    staleTime: 5 * 60_000,
    // A network error must not sign anyone out; keep retrying quietly.
    retry: (failures, error) => !(error instanceof ApiError && error.status === 401) && failures < 3,
  });

  // A session revoked elsewhere (or expired) turns any request into a 401.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearToken();
      queryClient.setQueryData(meQueryKey, null);
    });
    return () => setUnauthorizedHandler(null);
  }, [queryClient]);

  const user = me.data ?? null;
  // Keep showing the last known state while offline instead of bouncing to sign-in.
  const status: SessionStatus =
    me.data === undefined
      ? me.isError
        ? 'signedOut'
        : 'loading'
      : user === null
        ? 'signedOut'
        : user.username === null
          ? 'needsUsername'
          : 'signedIn';

  const value = useMemo(() => ({ status, user }), [status, user]);
  return <SessionContext value={value}>{children}</SessionContext>;
}

export function useSession(): SessionContextValue {
  const ctx = use(SessionContext);
  if (!ctx) throw new Error('useSession must be used inside <SessionProvider>');
  return ctx;
}

/** The signed-in user, for screens that are only reachable when signed in. */
export function useUser(): User {
  const { user } = useSession();
  if (!user) throw new Error('useUser needs a signed-in user');
  return user;
}
