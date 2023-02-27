import { DateTime, Interval } from "luxon";

export const zone = 'Europe/Rome';

const businessWeek = [1, 2, 3, 4, 5];

export function isBookeable(d: DateTime): boolean {
  const easter = getEasterDate(d.year);
  const mondayAfterEaster = easter.plus({ days: 1 });;
  if (mondayAfterEaster.month === d.month && mondayAfterEaster.day === d.day) return false;
  if (d.day === 25 && d.month === 4) return false;
  if (d.day === 2 && d.month === 6) return false;
  if (d.day === 15 && d.month === 8) return false;
  if (d.day === 1 && d.month === 11) return false;
  if (d.day === 1 && d.month === 1) return false;
  if (d.day === 1 && d.month === 1) return false;
  if (d.day === 25 && d.month === 12) return false;
  if (d.day === 19 && d.month === 9) return false;
  if (d.day === 1 && d.month === 5) return false;
  if (d.weekday === 6) return d.hour >= 7 && d.hour <= 11;
  return businessWeek.includes(d.weekday) && d.hour >= 7 && d.hour <= 21;
}

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
    locale: 'it',
  });
}

export function formatBooking(start: Date | string, end: Date | string | undefined = undefined, format: Intl.DateTimeFormatOptions): string {
  const s = typeof start === 'string' ? DateTime.fromISO(start) : DateTime.fromJSDate(start);
  const e = typeof end === 'string' ? DateTime.fromISO(end) : end ? DateTime.fromJSDate(end) : s.plus({ hours: 1 });

  return Interval.fromDateTimes(s, e).toLocaleString(format, {
    locale: 'it',
  });
}

function getEasterDate(year: number): DateTime {
  const f = Math.floor,
    // Golden Number - 1
    G = year % 19,
    C = f(year / 100),
    // related to Epact
    H = (C - f(C / 4) - f((8 * C + 13) / 25) + 19 * G + 15) % 30,
    // number of days from 21 March to the Paschal full moon
    I = H - f(H / 28) * (1 - f(29 / (H + 1)) * f((21 - G) / 11)),
    // weekday for the Paschal full moon
    J = (year + f(year / 4) + I + 2 - C + f(C / 4)) % 7,
    // number of days from 21 March to the Sunday on or before the Paschal full moon
    L = I - J,
    month = 3 + f((L + 40) / 44),
    day = L + 28 - 31 * f(month / 4);

  return DateTime.fromJSDate(new Date(year, month - 1, day))
}
