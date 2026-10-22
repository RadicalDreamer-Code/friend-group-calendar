import * as Haptics from 'expo-haptics';

/**
 * Light touch feedback on iOS and Android. Failures are ignored: haptics are
 * a nicety, and devices without a motor simply stay still.
 */
export const haptics = {
  /** A choice changed: a segment, a day, a color, an answer. */
  selection: () => Haptics.selectionAsync().catch(() => {}),
  /** Something was saved or created. */
  success: () => Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success).catch(() => {}),
  /** Before an action that deletes or removes something. */
  warning: () => Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning).catch(() => {}),
};
