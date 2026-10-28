import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { API_URL, api, unwrap } from './client';

export const tokensQueryKey = ['api-tokens'] as const;

export const tokenNameMaxLength = 40;
/** How many live tokens a user can have; the API refuses more. */
export const maxTokens = 10;
export type TokenExpiry = 30 | 90 | 365;

/** The MCP endpoint that access tokens are for. */
export const mcpUrl = `${API_URL}/mcp`;

/** The Claude Code command that adds this app's MCP server with a token. */
export function claudeCodeCommand(token: string) {
  return `claude mcp add --transport http friend-cal ${mcpUrl} --header "Authorization: Bearer ${token}"`;
}

export function useApiTokens() {
  return useQuery({
    queryKey: tokensQueryKey,
    queryFn: async () => (await unwrap(api.GET('/me/tokens'))).tokens,
  });
}

/** Resolves to the new token, including the secret that is shown only this once. */
export function useCreateApiToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, expiresInDays }: { name: string; expiresInDays: TokenExpiry }) =>
      unwrap(api.POST('/me/tokens', { body: { name, expiresInDays } })),
    onSettled: () => queryClient.invalidateQueries({ queryKey: tokensQueryKey }),
  });
}

export function useRevokeApiToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tokenId: string) => unwrap(api.DELETE('/me/tokens/{tokenId}', { params: { path: { tokenId } } })),
    onSettled: () => queryClient.invalidateQueries({ queryKey: tokensQueryKey }),
  });
}
