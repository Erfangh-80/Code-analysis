import { afterEach, describe, expect, it, vi } from "vitest";

import { apiRequest } from "@/services/api/client";

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("apiRequest", () => {
  it("preserves safe backend validation errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify({ error: { code: "VALIDATION_ERROR", message: "مسیر معتبر نیست." } }), { status: 400 })),
    );

    await expect(apiRequest("/analyses")).rejects.toMatchObject({
      status: 400,
      code: "VALIDATION_ERROR",
      message: "مسیر معتبر نیست.",
      retryable: false,
    });
  });

  it("turns a timed-out local request into a retryable API error", async () => {
    vi.useFakeTimers();
    vi.stubGlobal(
      "fetch",
      vi.fn((_input: RequestInfo | URL, init?: RequestInit) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")));
        }),
      ),
    );

    const request = apiRequest("/health", { timeoutMs: 25 });
    const timedOut = expect(request).rejects.toMatchObject({
      status: 408,
      code: "REQUEST_TIMEOUT",
      retryable: true,
    });
    await vi.advanceTimersByTimeAsync(25);
    await timedOut;
  });

  it("turns an unreachable local service into a retryable API error", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => { throw new TypeError("Failed to fetch"); }));

    await expect(apiRequest("/health")).rejects.toMatchObject({
      status: 0,
      code: "LOCAL_API_UNREACHABLE",
      retryable: true,
    });
  });
});
