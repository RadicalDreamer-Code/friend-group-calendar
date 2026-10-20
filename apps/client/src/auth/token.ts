import * as SecureStore from 'expo-secure-store';

/**
 * Native: the session token lives in the Keychain / Keystore and is sent as
 * a bearer token. It is cached in memory so requests can read it synchronously.
 */
const KEY = 'session-token';

let token: string | null = null;

export async function loadToken(): Promise<void> {
  try {
    token = await SecureStore.getItemAsync(KEY);
  } catch {
    token = null;
  }
}

export function getToken(): string | null {
  return token;
}

export async function saveToken(value: string): Promise<void> {
  token = value;
  await SecureStore.setItemAsync(KEY, value);
}

export async function clearToken(): Promise<void> {
  token = null;
  try {
    await SecureStore.deleteItemAsync(KEY);
  } catch {
    // Nothing stored, or storage unavailable: either way there is no token now.
  }
}
