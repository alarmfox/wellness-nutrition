import { DateTime, Interval } from "luxon";

export const defaultFormatOpts: Intl.DateTimeFormatOptions = {
  weekday: 'long',
  month: 'long',
  day: 'numeric',
  minute: '2-digit',
  hour: '2-digit',
  hour12: false,
}
export function formatDate(s: string | Date, format: Intl.DateTimeFormatOptions = defaultFormatOpts): string {
  const d = typeof s === 'string' ? DateTime.fromISO(s) : DateTime.fromJSDate(s);

  return d.toLocaleString(format, {
    locale: 'it'
  });
}

export function formatBooking(start: Date | string, end: Date | string | undefined = undefined, format: Intl.DateTimeFormatOptions): string {
  const s = typeof start === 'string' ? DateTime.fromISO(start) : DateTime.fromJSDate(start);
  const e = typeof end === 'string' ? DateTime.fromISO(end) : end ? DateTime.fromJSDate(end) : s.plus({ hours: 1 });

  return Interval.fromDateTimes(s, e).toLocaleString(format, {
    locale: 'it',
  });
}
