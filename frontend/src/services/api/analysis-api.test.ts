import { afterEach, describe, expect, it, vi } from "vitest";

import { analysisApi } from "@/services/api/analysis-api";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("analysisApi files", () => {
  it("serializes every files-table filter and sort as an API query", async () => {
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response(
          JSON.stringify({ items: [], page: 1, pageSize: 20, totalItems: 0 }),
          { status: 200 },
        ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await analysisApi.files("analysis_123", {
      page: 1,
      pageSize: 20,
      q: "src/api",
      language: ["go"],
      riskLevel: ["high", "medium"],
      analysisStatus: ["analyzed"],
      minScore: 50,
      maxScore: 90,
      sort: "score",
      direction: "desc",
    });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "/api/v1/analyses/analysis_123/files?page=1&pageSize=20&q=src%2Fapi&riskLevel=high&riskLevel=medium&language=go&analysisStatus=analyzed&minScore=50&maxScore=90&sort=score&direction=desc",
    );
  });
});
