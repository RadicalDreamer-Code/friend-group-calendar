type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel: string;
  destructive?: boolean;
};

/** Web: the browser's confirm dialog (react-native-web has no Alert). */
export async function confirm({ title, message }: ConfirmOptions): Promise<boolean> {
  return window.confirm(message ? `${title}\n\n${message}` : title);
}
