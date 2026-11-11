import { toJalaali } from "jalaali-js";

/** API dates stay UTC/RFC3339; this is presentation-only Persian calendar formatting. */
export function formatJalali(iso?: string | null) {
  if (!iso) return "—";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  const j = toJalaali(date.getUTCFullYear(), date.getUTCMonth() + 1, date.getUTCDate());
  return `${j.jy}/${String(j.jm).padStart(2, "0")}/${String(j.jd).padStart(2, "0")}`;
}
// datetime-local has no offset. The form explicitly labels its values as UTC,
// so append Z instead of silently interpreting the browser's local timezone.
export function toUTCISOString(value: string) {
  if (!value) return undefined;
  const normalized = value.length === 16 ? `${value}:00Z` : `${value}Z`;
  return new Date(normalized).toISOString();
}
