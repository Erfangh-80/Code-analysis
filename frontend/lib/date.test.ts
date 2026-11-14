import { describe, expect, it } from "vitest";
import { formatJalali, toUTCISOString } from "./date";
describe("formatJalali", () => { it("renders a known Nowruz date in the Persian calendar", () => { expect(formatJalali("2025-03-21T00:00:00Z")).toBe("1404/01/01"); }); it("does not invent a date for missing input", () => { expect(formatJalali()).toBe("—"); }); it("treats form datetime values as explicitly UTC", () => { expect(toUTCISOString("2025-03-21T12:30")).toBe("2025-03-21T12:30:00.000Z"); }); });
