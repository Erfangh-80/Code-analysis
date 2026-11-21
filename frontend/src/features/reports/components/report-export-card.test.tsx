import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ReportExportCard } from "@/features/reports/components/report-export-card";

const exportMock = vi.fn();

vi.mock("@/services/api/analysis-api", () => ({
  analysisApi: { export: (...args: unknown[]) => exportMock(...args) },
}));

afterEach(() => {
  exportMock.mockReset();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

const report = {
  analysis: { completedAt: "2026-11-21T00:00:00Z" },
  artifacts: {
    static: { files: [] },
    risk: { files: [], profile: { version: "risk-v1" } },
  },
} as never;

describe("ReportExportCard", () => {
  it("starts a JSON report download after the user reviewed the summary", async () => {
    exportMock.mockResolvedValue({ blob: new Blob(["{}"], { type: "application/json" }), contentDisposition: 'attachment; filename="report.json"' });
    const objectURL = vi.fn(() => "blob:report");
    vi.stubGlobal("URL", { createObjectURL: objectURL, revokeObjectURL: vi.fn() });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);

    render(<ReportExportCard analysisId="analysis_1" report={report} />);
    fireEvent.click(screen.getByRole("button", { name: /دریافت گزارش JSON/i }));

    expect(await screen.findByText("دانلود گزارش JSON آغاز شد.")).toBeInTheDocument();
    expect(exportMock).toHaveBeenCalledWith("analysis_1", "json", { timeoutMs: 60_000 });
    expect(objectURL).toHaveBeenCalled();
    expect(click).toHaveBeenCalled();
  });

  it("shows a safe export error without leaving the report page", async () => {
    exportMock.mockRejectedValue(new Error("network"));

    render(<ReportExportCard analysisId="analysis_1" report={report} />);
    fireEvent.click(screen.getByRole("button", { name: /دریافت فهرست CSV/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("امکان دریافت گزارش وجود ندارد");
  });
});
