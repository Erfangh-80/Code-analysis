import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ThemeController } from "@/app/theme-controller";
import { DashboardLayout } from "@/components/templates/dashboard-layout";
import { CoreAnalysisPage } from "@/features/core/pages/core-analysis-page";
import { useUIStore } from "@/store/ui-store";

afterEach(() => {
  useUIStore.getState().setTheme("light");
  document.documentElement.removeAttribute("data-theme");
  document.documentElement.style.removeProperty("color-scheme");
  localStorage.clear();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("core analysis workspace", () => {
  it("renders the Persian core-analysis workflow and checks the local API relatively", async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      if (String(input).endsWith("/health")) {
        return new Response(JSON.stringify({ status: "ok", version: "0.1.0-core" }), { status: 200 });
      }
      return new Response(JSON.stringify({ files: [], modules: [] }), { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const router = createMemoryRouter([{ element: <DashboardLayout />, children: [{ index: true, element: <CoreAnalysisPage /> }] }]);

    render(<QueryClientProvider client={client}><ThemeController /><RouterProvider router={router} /></QueryClientProvider>);

    expect(screen.getByRole("heading", { name: "تحلیل‌گر ریسک کد" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "۱. تحلیل ایستای فایل‌ها" })).toBeInTheDocument();
    expect(await screen.findByText(/سرویس محلی متصل است/i)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/health", expect.any(Object));
  });
});
