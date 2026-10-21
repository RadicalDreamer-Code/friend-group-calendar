/** "just now", "3 minutes ago", "2 hours ago", "4 days ago", "Oct 12". */
export function timeAgo(iso: string, now = Date.now()): string {
  const minutes = Math.round((now - new Date(iso).getTime()) / 60_000);
  if (minutes < 5) return 'just now';
  if (minutes < 60) return `${minutes} minutes ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} ${hours === 1 ? 'hour' : 'hours'} ago`;
  const days = Math.round(hours / 24);
  if (days < 7) return `${days} ${days === 1 ? 'day' : 'days'} ago`;
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

/** "Active now", "Active 3 hours ago", "Active Oct 12". */
export function lastActive(iso: string, now = Date.now()): string {
  const ago = timeAgo(iso, now);
  return ago === 'just now' ? 'Active now' : `Active ${ago}`;
}

/** "October 2026". */
export function monthYear(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}
