import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { CoreAnalysisPage } from "@/features/core/pages/core-analysis-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("CoreAnalysisPage", () => {
  it("shows static results returned by the local backend", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith("/static-analysis")) {
        return new Response(JSON.stringify({
          files: [{ path: "src/main.go", language: "go", status: "analyzed", usedFallback: false, metrics: { linesOfCode: 12, functionCount: 1, classCount: 0, functionLength: { max: 4, total: 4 }, classLength: { max: 0, total: 0 } }, functions: [], classes: [], diagnostics: [] }],
          modules: [{ path: "src", fileCount: 1, failedFileCount: 0, metrics: { linesOfCode: 12, functionCount: 1, classCount: 0, functionLength: { max: 4, total: 4 }, classLength: { max: 0, total: 0 } } }],
        }), { status: 200 });
      }
      return new Response(JSON.stringify({ status: "ok", version: "0.1.0-core" }), { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });

    render(<QueryClientProvider client={client}><CoreAnalysisPage /></QueryClientProvider>);
    fireEvent.change(screen.getByLabelText("مسیر مطلق ریشهٔ پروژه"), { target: { value: "/workspace/project" } });
    fireEvent.change(screen.getByLabelText("فایل‌های نسبی"), { target: { value: "src/main.go" } });
    fireEvent.click(screen.getByRole("button", { name: "تحلیل ایستا" }));

    expect(await screen.findByText("src/main.go")).toBeInTheDocument();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/v1/static-analysis", expect.any(Object)));
  });
});
