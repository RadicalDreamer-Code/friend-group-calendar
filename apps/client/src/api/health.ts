import { useQuery } from '@tanstack/react-query';

import { api } from './client';

export type ServerStatus = 'checking' | 'connected' | 'degraded' | 'unreachable';

/** Polls GET /healthz so the UI can show whether the API and database are up. */
export function useServerStatus(): ServerStatus {
  const query = useQuery({
    queryKey: ['health'],
    queryFn: async () => {
      // A 503 still carries a Health body, so read it from either branch.
      const { data, error } = await api.GET('/healthz');
      const body = data ?? error;
      if (!body) throw new Error('empty health response');
      return body;
    },
    refetchInterval: 30_000,
    retry: false,
  });

  if (query.isPending) return 'checking';
  if (query.isError) return 'unreachable';
  return query.data.status === 'ok' ? 'connected' : 'degraded';
}
