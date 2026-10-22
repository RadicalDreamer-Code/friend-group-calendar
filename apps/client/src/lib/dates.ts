/**
 * Whole-day dates as the API sends them: "2026-10-31", and months as
 * "2026-10". They are plain calendar days with no time zone, so they are
 * only ever turned into a Date at local midnight, for formatting.
 */

const pad = (n: number) => String(n).padStart(2, '0');

/** The date as YYYY-MM-DD in the device's time zone. */
export function toISODate(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function today(): string {
  return toISODate(new Date());
}

/** Local midnight of a YYYY-MM-DD date. */
export function fromISODate(date: string): Date {
  const [y, m, d] = date.split('-').map(Number);
  return new Date(y, m - 1, d);
}

/** "2026-10" for "2026-10-31". */
export function monthOf(date: string): string {
  return date.slice(0, 7);
}

/** The month `n` months after (or before) a YYYY-MM month. */
export function addMonths(month: string, n: number): string {
  const [y, m] = month.split('-').map(Number);
  return toISODate(new Date(y, m - 1 + n, 1)).slice(0, 7);
}

/** Every day of a YYYY-MM month as YYYY-MM-DD. */
export function daysOfMonth(month: string): string[] {
  const [y, m] = month.split('-').map(Number);
  const count = new Date(y, m, 0).getDate();
  return Array.from({ length: count }, (_, i) => `${month}-${pad(i + 1)}`);
}

/** Entries can be planned from 2000 to 2099, like the API allows. */
export const firstMonth = '2000-01';
export const lastMonth = '2099-12';

/** True for a real YYYY-MM-DD day between firstMonth and lastMonth, e.g. from a URL. */
export function isISODate(s: string | undefined): s is string {
  if (!s || !/^\d{4}-\d{2}-\d{2}$/.test(s)) return false;
  const month = monthOf(s);
  return month >= firstMonth && month <= lastMonth && daysOfMonth(month).includes(s);
}

/**
 * The weekday weeks start on, 0 = Sunday, from the device's locale where the
 * platform tells us, else Monday.
 */
export const firstDayOfWeek: number = (() => {
  try {
    const locale = new Intl.Locale(Intl.DateTimeFormat().resolvedOptions().locale) as Intl.Locale & {
      getWeekInfo?: () => { firstDay: number };
      weekInfo?: { firstDay: number };
    };
    const firstDay = (locale.getWeekInfo?.() ?? locale.weekInfo)?.firstDay;
    if (firstDay) return firstDay % 7; // weekInfo counts Monday = 1 ... Sunday = 7
  } catch {
    // Intl.Locale or weekInfo missing: fall through.
  }
  return 1;
})();

/** Short weekday names in display order, e.g. ["Mon", "Tue", …]. */
export function weekdayNames(format: 'narrow' | 'short' = 'short'): string[] {
  // 2023-01-01 was a Sunday.
  return Array.from({ length: 7 }, (_, i) =>
    new Date(2023, 0, 1 + ((firstDayOfWeek + i) % 7)).toLocaleDateString(undefined, { weekday: format }),
  );
}

/** How many empty cells come before the 1st in a month grid. */
export function leadingBlanks(month: string): number {
  return (fromISODate(`${month}-01`).getDay() - firstDayOfWeek + 7) % 7;
}

/** "October 2026". */
export function formatMonth(month: string): string {
  return fromISODate(`${month}-01`).toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

/** "Saturday, October 31", with the year when it is not this year. */
export function formatDay(date: string, now = today()): string {
  const sameYear = date.slice(0, 4) === now.slice(0, 4);
  return fromISODate(date).toLocaleDateString(undefined, {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
    ...(sameYear ? {} : { year: 'numeric' }),
  });
}

/** "Today", "Tomorrow", "Yesterday" or formatDay. */
export function relativeDay(date: string, now = today()): string {
  const base = fromISODate(now);
  const days = Math.round((fromISODate(date).getTime() - base.getTime()) / 86_400_000);
  if (days === 0) return 'Today';
  if (days === 1) return 'Tomorrow';
  if (days === -1) return 'Yesterday';
  return formatDay(date, now);
}
