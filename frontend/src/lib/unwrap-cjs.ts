/**
 * Unwraps the default export of a CommonJS module regardless of the bundler's
 * interop: Vite dev pre-bundling may yield the exports namespace object while
 * Vitest/Node yields the default export directly. Values that are already the
 * real export (functions, forwardRef objects, plain data without a `default`
 * key) pass through untouched.
 */
export function unwrapCjs<T>(module: T): T {
  const boxed = module as { default?: T } | null;
  if (boxed && typeof boxed === "object" && "default" in boxed && boxed.default) {
    return boxed.default;
  }
  return module;
}
