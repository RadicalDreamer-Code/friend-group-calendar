import createClient from 'openapi-fetch';

import type { paths } from './schema';

/**
 * Base URL of the Go API. Set EXPO_PUBLIC_API_URL to change it; on a phone
 * use your computer's LAN address instead of localhost.
 */
export const API_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';

/** Typed client generated from packages/api/openapi.yaml (`npm run gen:api`). */
export const api = createClient<paths>({
  baseUrl: API_URL,
  // Sends the session cookie on web once sign-in exists.
  credentials: 'include',
});
