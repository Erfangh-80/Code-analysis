import { describe, expect, it } from "vitest";

import { unwrapCjs } from "@/lib/unwrap-cjs";

describe("unwrapCjs", () => {
  it("returns the default export when given a CJS exports namespace", () => {
    const component = () => null;
    const namespace = { default: component, __esModule: true };
    expect(unwrapCjs(namespace)).toBe(component);
  });

  it("passes functions through untouched", () => {
    const component = () => null;
    expect(unwrapCjs(component)).toBe(component);
  });

  it("passes plain data objects without a default key through untouched", () => {
    const calendar = { name: "persian", startYear: 1 };
    expect(unwrapCjs(calendar)).toBe(calendar);
  });

  it("passes nullish values through untouched", () => {
    expect(unwrapCjs(null)).toBe(null);
  });
});
