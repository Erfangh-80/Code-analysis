import DateObject from "react-date-object";
import { describe, expect, it } from "vitest";

import { emptyGitResultCauses, toIsoUtc, validateHistoryRange } from "@/lib/datetime";

describe("toIsoUtc", () => {
  it("converts a picker value to a Gregorian UTC ISO instant", () => {
    const value = new DateObject(new Date(Date.UTC(2026, 0, 15, 10, 30, 0)));
    expect(toIsoUtc(value)).toBe("2026-01-15T10:30:00.000Z");
  });

  it("returns undefined when no date is selected", () => {
    expect(toIsoUtc(null)).toBeUndefined();
  });
});

describe("validateHistoryRange", () => {
  const now = Date.UTC(2026, 5, 15, 12, 0, 0);

  it("rejects a range whose start is after its end", () => {
    const from = new DateObject(new Date(Date.UTC(2026, 4, 10)));
    const to = new DateObject(new Date(Date.UTC(2026, 4, 1)));
    const result = validateHistoryRange(from, to, now);
    expect(result.error).toContain("«از تاریخ»");
  });

  it("warns when the start date lies in the future", () => {
    const from = new DateObject(new Date(now + 86_400_000));
    const result = validateHistoryRange(from, null, now);
    expect(result.error).toBeUndefined();
    expect(result.warnings.some((warning) => warning.includes("آینده"))).toBe(true);
  });

  it("accepts a valid past range without warnings", () => {
    const from = new DateObject(new Date(Date.UTC(2026, 0, 1)));
    const to = new DateObject(new Date(Date.UTC(2026, 1, 1)));
    const result = validateHistoryRange(from, to, now);
    expect(result.error).toBeUndefined();
    expect(result.warnings).toHaveLength(0);
  });

  it("accepts an empty range (full history)", () => {
    const result = validateHistoryRange(null, null, now);
    expect(result.error).toBeUndefined();
    expect(result.warnings).toHaveLength(0);
  });
});

describe("emptyGitResultCauses", () => {
  it("always explains the empty-range possibility", () => {
    expect(emptyGitResultCauses(false).length).toBeGreaterThanOrEqual(2);
  });

  it("adds a range-ordering hint when a range was selected", () => {
    const withRange = emptyGitResultCauses(true);
    const withoutRange = emptyGitResultCauses(false);
    expect(withRange.length).toBeGreaterThan(withoutRange.length);
  });
});
