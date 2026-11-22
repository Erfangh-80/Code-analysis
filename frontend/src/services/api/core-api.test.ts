import { afterEach, describe, expect, it, vi } from "vitest";

import { coreApi } from "@/services/api/core-api";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("coreApi", () => {
  it("sends static-analysis input only to the local direct endpoint", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ files: [], modules: [] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await coreApi.staticAnalysis({ rootPath: "/workspace/project", files: ["src/main.go"] });

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/static-analysis", expect.objectContaining({ method: "POST" }));
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({ rootPath: "/workspace/project", files: ["src/main.go"] });
  });

  it("posts complete backend-owned risk inputs without calculating a score in the browser", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ files: [], modules: [], heatmap: { cells: [] } }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await coreApi.riskAnalysis({
      availableMetrics: ["lines_of_code", "max_function_length", "max_class_length"],
      files: [{ path: "src/main.go", metrics: { linesOfCode: 12, maxFunctionLength: 4, maxClassLength: 0, gitChurn: 0, changeFrequency: 0 } }],
    });

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/risk-analysis", expect.objectContaining({ method: "POST" }));
  });
});
