import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { CoreAnalysisPage } from "@/features/core/pages/core-analysis-page";
import type { FileAnalysis } from "@/types/api";

afterEach(() => {
  vi.unstubAllGlobals();
});

function fileAnalysis(index: number): FileAnalysis {
  return {
    path: `src/file-${String(index).padStart(2, "0")}.go`,
    language: "go",
    status: "analyzed",
    usedFallback: false,
    metrics: { linesOfCode: 10 + index, functionCount: 1, classCount: 0, functionLength: { max: 4, total: 4 }, classLength: { max: 0, total: 0 } },
    functions: [],
    classes: [],
    diagnostics: [],
  };
}

function renderPage(fetchMock: ReturnType<typeof vi.fn>) {
  vi.stubGlobal("fetch", fetchMock);
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}><CoreAnalysisPage /></QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("CoreAnalysisPage", () => {
  it("discovers files from the root path alone and shows returned results", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("/static-analysis")) {
        expect(JSON.parse(String(init?.body))).toEqual({ rootPath: "/workspace/project" });
        return new Response(JSON.stringify({
          files: [fileAnalysis(1)],
          modules: [{ path: "src", fileCount: 1, failedFileCount: 0, metrics: { linesOfCode: 11, functionCount: 1, classCount: 0, functionLength: { max: 4, total: 4 }, classLength: { max: 0, total: 0 } } }],
        }), { status: 200 });
      }
      return new Response(JSON.stringify({ status: "ok", version: "0.1.0-core" }), { status: 200 });
    });
    renderPage(fetchMock);

    fireEvent.change(screen.getByLabelText("مسیر مطلق ریشهٔ پروژه"), { target: { value: "/workspace/project" } });
    fireEvent.click(screen.getByRole("button", { name: "تحلیل ایستا" }));

    expect(await screen.findByText("src/file-01.go")).toBeInTheDocument();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith("/api/v1/static-analysis", expect.any(Object)));
  });

  it("paginates large static results", async () => {
    const files = Array.from({ length: 12 }, (_, index) => fileAnalysis(index + 1));
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith("/static-analysis")) {
        return new Response(JSON.stringify({ files, modules: [] }), { status: 200 });
      }
      return new Response(JSON.stringify({ status: "ok", version: "0.1.0-core" }), { status: 200 });
    });
    renderPage(fetchMock);

    fireEvent.change(screen.getByLabelText("مسیر مطلق ریشهٔ پروژه"), { target: { value: "/workspace/project" } });
    fireEvent.click(screen.getByRole("button", { name: "تحلیل ایستا" }));

    expect(await screen.findByText("src/file-01.go")).toBeInTheDocument();
    expect(screen.queryByText("src/file-12.go")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "صفحهٔ بعدی" }));

    expect(await screen.findByText("src/file-12.go")).toBeInTheDocument();
    expect(screen.queryByText("src/file-01.go")).not.toBeInTheDocument();
  });

  it("renders Jalali date inputs for the Git history range", () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({ status: "ok", version: "0.1.0-core" }), { status: 200 }));
    renderPage(fetchMock);

    expect(screen.getByText("از تاریخ")).toBeInTheDocument();
    expect(screen.getByText("تا تاریخ")).toBeInTheDocument();
  });

  it("opens a detail dialog with the file's real metrics", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith("/static-analysis")) {
        return new Response(JSON.stringify({
          files: [fileAnalysis(1)],
          modules: [],
        }), { status: 200 });
      }
      return new Response(JSON.stringify({ status: "ok", version: "0.1.0-core" }), { status: 200 });
    });
    renderPage(fetchMock);

    fireEvent.change(screen.getByLabelText("مسیر مطلق ریشهٔ پروژه"), { target: { value: "/workspace/project" } });
    fireEvent.click(screen.getByRole("button", { name: "تحلیل ایستا" }));
    expect(await screen.findByText("src/file-01.go")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "جزئیات" }));

    expect(await screen.findByText("بیشترین طول تابع")).toBeInTheDocument();
    expect(screen.getByText("مجموع طول توابع")).toBeInTheDocument();
  });
});
