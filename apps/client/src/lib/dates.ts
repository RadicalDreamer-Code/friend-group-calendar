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

/** Whole days from `now` to `date`: 2 for the day after tomorrow, -1 for yesterday. */
export function daysFrom(date: string, now = today()): number {
  // Rounded, since a day with a daylight saving change is 23 or 25 hours long.
  return Math.round((fromISODate(date).getTime() - fromISODate(now).getTime()) / 86_400_000);
}

/** "today", "tomorrow" or "yesterday", or null for any other day. */
function nearDay(days: number): string | null {
  return days === 0 ? 'today' : days === 1 ? 'tomorrow' : days === -1 ? 'yesterday' : null;
}

export const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

/** "today", "tomorrow", "in 2 days", "yesterday", "3 days ago": for chips and second lines. */
export function inDays(date: string, now = today()): string {
  const days = daysFrom(date, now);
  return nearDay(days) ?? (days > 0 ? `in ${days} days` : `${-days} days ago`);
}

/** "Today", "Tomorrow", "2 days to go", "Yesterday", "3 days ago": an entry's countdown. */
export function countdown(date: string, now = today()): string {
  const days = daysFrom(date, now);
  return capitalize(nearDay(days) ?? (days > 0 ? `${days} days to go` : `${-days} days ago`));
}

/** "Sat, Oct 31", with the year when it is not this year. */
export function formatShortDay(date: string, now = today()): string {
  const sameYear = date.slice(0, 4) === now.slice(0, 4);
  return fromISODate(date).toLocaleDateString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    ...(sameYear ? {} : { year: 'numeric' }),
  });
}

/** "Today", "Tomorrow", "Yesterday" or formatDay. */
export function relativeDay(date: string, now = today()): string {
  const near = nearDay(daysFrom(date, now));
  return near ? capitalize(near) : formatDay(date, now);
}

/** The YYYY-MM-DD day `n` days after (or before) a day. */
export function addDays(date: string, n: number): string {
  const d = fromISODate(date);
  return toISODate(new Date(d.getFullYear(), d.getMonth(), d.getDate() + n));
}

/** How many days a stretch from `start` to `end` covers, both included: 4 for Oct 26 to Oct 29. */
export function spanDays(start: string, end: string): number {
  return daysFrom(end, start) + 1;
}

/** "Day 2 of 4": which day of a stretch `date` is. */
export function dayOf(start: string, end: string, date: string): string {
  return `Day ${daysFrom(date, start) + 1} of ${spanDays(start, end)}`;
}

/**
 * A trip's countdown: "3 days to go" and "Tomorrow" up to its first day,
 * then "Day 2 of 4", then "Ended".
 */
export function tripCountdown(start: string, end: string, now = today()): string {
  if (now < start) return countdown(start, now);
  if (now <= end) return dayOf(start, end, now);
  return 'Ended';
}

/**
 * "Mon 26 → Thu 29 Oct", "Fri 30 Oct → Mon 2 Nov", with the year on each end
 * when the two are in different years or not this year.
 */
export function formatRange(start: string, end: string, now = today()): string {
  const withYear = start.slice(0, 4) !== end.slice(0, 4) || start.slice(0, 4) !== now.slice(0, 4);
  const sameMonth = monthOf(start) === monthOf(end);
  const part = (date: string, showMonth: boolean) => {
    const d = fromISODate(date);
    return [
      d.toLocaleDateString(undefined, { weekday: 'short' }),
      d.getDate(),
      showMonth ? d.toLocaleDateString(undefined, { month: 'short' }) : null,
      withYear && showMonth ? d.getFullYear() : null,
    ]
      .filter((p) => p !== null)
      .join(' ');
  };
  return `${part(start, !sameMonth || withYear)} → ${part(end, true)}`;
}
