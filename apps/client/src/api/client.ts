import createClient from 'openapi-fetch';

import { getToken } from '@/auth/token';

import type { components, paths } from './schema';

export type User = components['schemas']['User'];
export type Session = components['schemas']['Session'];
export type Platform = components['schemas']['Platform'];
export type Person = components['schemas']['Person'];
export type FriendshipStatus = components['schemas']['FriendshipStatus'];
export type Friend = components['schemas']['Friend'];
export type FriendRequest = components['schemas']['FriendRequest'];
export type Role = components['schemas']['Role'];
export type MemberRole = components['schemas']['MemberRole'];
export type CalendarSummary = components['schemas']['CalendarSummary'];
export type Calendar = components['schemas']['Calendar'];
export type CalendarMember = components['schemas']['CalendarMember'];
export type Entry = components['schemas']['Entry'];
export type EntryColor = components['schemas']['EntryColor'];
export type EntryCover = components['schemas']['EntryCover'];
export type EntryResponseStatus = components['schemas']['EntryResponseStatus'];

/**
 * Base URL of the Go API. Set EXPO_PUBLIC_API_URL to change it; on a phone
 * use your computer's LAN address instead of localhost.
 */
export const API_URL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';

/** Typed client generated from packages/api/openapi.yaml (`npm run gen:api`). */
export const api = createClient<paths>({
  baseUrl: API_URL,
  // Sends the session cookie on web.
  credentials: 'include',
});

let onUnauthorized: (() => void) | null = null;

/** Called when the API rejects the session, e.g. after it was revoked on another device. */
export function setUnauthorizedHandler(handler: (() => void) | null) {
  onUnauthorized = handler;
}

api.use({
  onRequest({ request }) {
    const token = getToken();
    if (token) request.headers.set('Authorization', `Bearer ${token}`);
    return request;
  },
  onResponse({ response, schemaPath }) {
    if (response.status === 401 && !schemaPath.startsWith('/auth/')) onUnauthorized?.();
  },
});

/** An error response from the API, with the stable `code` to switch on. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

type Result<T> = { data?: T; error?: unknown; response: Response };

/** Returns the data of a successful response or throws an ApiError. */
export async function unwrap<T>(request: Promise<Result<T>>): Promise<T> {
  let result: Result<T>;
  try {
    result = await request;
  } catch {
    throw new ApiError(0, 'network', 'Could not reach the server. Check your connection and try again.');
  }
  const { data, error, response } = result;
  if (response.ok) return data as T;
  const body = (error as { error?: { code?: string; message?: string } } | undefined)?.error;
  throw new ApiError(
    response.status,
    body?.code ?? 'unknown',
    body?.message ?? 'Something went wrong. Please try again.',
  );
}

/** True when the request never got an answer: offline, DNS, the server down. */
export function isNetworkError(error: unknown): boolean {
  return error instanceof ApiError && error.code === 'network';
}

/** A message to show the user for any error thrown by a query or mutation. */
export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return 'Something went wrong. Please try again.';
}
