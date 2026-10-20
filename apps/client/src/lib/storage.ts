import AsyncStorage from '@react-native-async-storage/async-storage';
import { Platform } from 'react-native';

/**
 * Small key-value store for per-device preferences (not for secrets).
 * On web it reads synchronously so the first render already has the value,
 * which avoids a flash of the wrong theme.
 */
export const storage = {
  getSync(key: string): string | null {
    if (Platform.OS !== 'web') return null;
    try {
      return globalThis.localStorage?.getItem(key) ?? null;
    } catch {
      return null;
    }
  },

  async get(key: string): Promise<string | null> {
    try {
      return await AsyncStorage.getItem(key);
    } catch {
      return null;
    }
  },

  async set(key: string, value: string): Promise<void> {
    try {
      await AsyncStorage.setItem(key, value);
    } catch {
      // Preferences are a convenience; failing to save one is not an error for the user.
    }
  },
};
