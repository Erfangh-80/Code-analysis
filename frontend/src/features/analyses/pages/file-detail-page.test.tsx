import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { FileDetailPage } from "@/features/analyses/pages/file-detail-page";

vi.mock("@/hooks/use-analysis-queries", () => ({
  useAnalysisFileQuery: () => ({
    isPending: false,
    isError: false,
    refetch: vi.fn(),
    data: {
      analysis: {
        path: "internal/api/server.go",
        language: "go",
        status: "analyzed",
        usedFallback: false,
        metrics: {
          linesOfCode: 120,
          functionCount: 2,
          classCount: 0,
          functionLength: { max: 40, total: 60 },
          classLength: { max: 0, total: 0 },
        },
        functions: null,
        classes: null,
        diagnostics: null,
      },
      git: {
        path: "internal/api/server.go",
        previousPaths: null,
        deleted: false,
        gitChurn: 12,
        changeFrequency: 2,
        authorCount: 1,
        firstChangedAt: null,
        lastChangedAt: null,
        changeConcentration: 0.5,
        changeDispersion: 0.5,
      },
      risk: {
        path: "internal/api/server.go",
        module: "internal/api",
        metrics: {},
        score: 77,
        level: "high",
        rank: 1,
        contributions: null,
      },
    },
  }),
}));

describe("FileDetailPage", () => {
  it("renders safely when a persisted Go report has null collection fields", () => {
    const router = createMemoryRouter(
      [{ path: "/analyses/:analysisId/files/detail", element: <FileDetailPage /> }],
      { initialEntries: ["/analyses/run-1/files/detail?path=internal%2Fapi%2Fserver.go"] },
    );

    render(<RouterProvider router={router} />);

    expect(screen.getByRole("heading", { name: "internal/api/server.go" })).toBeInTheDocument();
    expect(screen.getByText("مسیرهای پیشین")).toBeInTheDocument();
    expect(screen.getByText("سهم شاخصی ثبت نشده است")).toBeInTheDocument();
  });
});
