/**
 * Web: the session is an HttpOnly cookie that scripts cannot read, so there
 * is no token to keep. The browser sends the cookie with every API request.
 */
export async function loadToken(): Promise<void> {}

export function getToken(): string | null {
  return null;
}

export async function saveToken(_value: string): Promise<void> {}

export async function clearToken(): Promise<void> {}
