/**
 * Every time Candella shows is rendered in the workspace's IANA timezone with the
 * Pakistani en-PK format, so a receipt, the sales history and the stock ledger
 * agree with each other and with the shop's wall clock — whatever timezone the
 * device or server doing the rendering happens to be in.
 */

/** Used where no workspace is in scope, such as the platform console. */
export const DEFAULT_TIMEZONE = "Asia/Karachi";

const LOCALE = "en-PK";

/** Current calendar day at a workspace, independent of the device timezone. */
export function businessToday(timeZone: string): string {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  const get = (type: string) =>
    parts.find((part) => part.type === type)?.value ?? "";
  return `${get("year")}-${get("month")}-${get("day")}`;
}

/** "15-Oct-2026, 2:05 am" */
export function formatDateTime(
  iso: string,
  timeZone: string = DEFAULT_TIMEZONE,
): string {
  return new Intl.DateTimeFormat(LOCALE, {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone,
  }).format(new Date(iso));
}

/** "15-Oct-2026" */
export function formatDate(
  iso: string,
  timeZone: string = DEFAULT_TIMEZONE,
): string {
  return new Intl.DateTimeFormat(LOCALE, {
    dateStyle: "medium",
    timeZone,
  }).format(new Date(iso));
}
