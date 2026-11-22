import DateObjectImport from "react-date-object";
import type { DateObject } from "react-multi-date-picker";
import persianCalendarImport from "react-date-object/calendars/persian";
import persianFaLocaleImport from "react-date-object/locales/persian_fa";

import { unwrapCjs } from "@/lib/unwrap-cjs";

const DateObjectCtor = unwrapCjs(DateObjectImport);
const persian = unwrapCjs(persianCalendarImport);
const persian_fa = unwrapCjs(persianFaLocaleImport);

/** Renders an ISO/Gregorian instant back in the Jalali calendar for display. */
export function formatJalali(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new DateObjectCtor({ date: value, calendar: persian, locale: persian_fa });
  return date.isValid ? date.format("YYYY/MM/DD HH:mm") : "—";
}

/**
 * Converts a Jalali-picker value to the Gregorian RFC 3339 instant the API
 * expects. The UI keeps showing Jalali; only transport/storage uses UTC.
 */
export function toIsoUtc(value: DateObject | null): string | undefined {
  if (!value || !value.isValid) return undefined;
  return value.toDate().toISOString();
}

export interface HistoryRangeValidation {
  /** Blocking problem, e.g. «از تاریخ» بعد از «تا تاریخ». */
  error?: string;
  /** Non-blocking hints shown before submitting. */
  warnings: string[];
}

/** Validates the selected Jalali range before it is sent to the local API. */
export function validateHistoryRange(from: DateObject | null, to: DateObject | null, now: number = Date.now()): HistoryRangeValidation {
  const fromMs = from && from.isValid ? from.toDate().getTime() : undefined;
  const toMs = to?.isValid ? to.toDate().getTime() : undefined;
  const warnings: string[] = [];

  if (fromMs !== undefined && toMs !== undefined && fromMs > toMs) {
    return { error: "«از تاریخ» باید قبل از «تا تاریخ» باشد؛ بازهٔ فعلی هیچ لحظه‌ای را پوشش نمی‌دهد.", warnings };
  }
  if (fromMs !== undefined && fromMs > now) {
    warnings.push("«از تاریخ» در آینده است؛ طبیعتاً هیچ commitی بعد از الان وجود ندارد.");
  }
  if (toMs !== undefined && toMs > now) {
    warnings.push("«تا تاریخ» در آینده است؛ عملاً تا لحظهٔ فعلی تحلیل می‌شود.");
  }
  return { warnings };
}

/** Explains why a Git analysis may have returned an empty result. */
export function emptyGitResultCauses(hasRange: boolean): string[] {
  const causes = [
    "مخزن در بازهٔ انتخاب‌شده هیچ commitی ندارد؛ بازه را عریض‌تر کنید یا تاریخ‌ها را بررسی کنید.",
    "ممکن است مخزن تازه ساخته شده یا خالی باشد.",
  ];
  if (hasRange) {
    causes.push("اگر تاریخ «از» بعد از آخرین commit یا تاریخ «تا» قبل از نخستین commit است، نتیجه خالی می‌ماند.");
  }
  return causes;
}
