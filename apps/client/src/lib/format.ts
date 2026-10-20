/** "Active now", "Active 3 hours ago", "Active Oct 12". */
export function lastActive(iso: string, now = Date.now()): string {
  const minutes = Math.round((now - new Date(iso).getTime()) / 60_000);
  if (minutes < 5) return 'Active now';
  if (minutes < 60) return `Active ${minutes} minutes ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `Active ${hours} ${hours === 1 ? 'hour' : 'hours'} ago`;
  const days = Math.round(hours / 24);
  if (days < 7) return `Active ${days} ${days === 1 ? 'day' : 'days'} ago`;
  return `Active ${new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`;
}

/** "October 2026". */
export function monthYear(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}
