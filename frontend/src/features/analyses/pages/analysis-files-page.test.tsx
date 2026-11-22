import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AnalysisFilesPage } from "@/features/analyses/pages/analysis-files-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("AnalysisFilesPage", () => {
  it("sends score filtering to the files API instead of downloading and filtering the full report", async () => {
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response(
          JSON.stringify({ items: [], page: 1, pageSize: 20, totalItems: 0 }),
          { status: 200 },
        ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const router = createMemoryRouter(
      [{ path: "/analyses/:analysisId/files", element: <AnalysisFilesPage /> }],
      { initialEntries: ["/analyses/analysis_123/files"] },
    );
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    fireEvent.change(screen.getByLabelText("حداقل امتیاز ریسک"), {
      target: { value: "50" },
    });

    await waitFor(() =>
      expect(
        fetchMock.mock.calls.some(
          ([url]) =>
            String(url) ===
            "/api/v1/analyses/analysis_123/files?page=1&pageSize=20&minScore=50&sort=path&direction=asc",
        ),
      ).toBe(true),
    );
    expect(
      fetchMock.mock.calls.some(([url]) => String(url).includes("/report")),
    ).toBe(false);
  });
});
