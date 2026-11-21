import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ThemeController } from "@/app/theme-controller";
import { DashboardLayout } from "@/components/templates/dashboard-layout";
import { DashboardPage } from "@/features/dashboard/pages/dashboard-page";
import { useUIStore } from "@/store/ui-store";

afterEach(() => {
  useUIStore.getState().setTheme("light");
  document.documentElement.removeAttribute("data-theme");
  document.documentElement.style.removeProperty("color-scheme");
  localStorage.clear();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("dashboard shell", () => {
  it("renders Persian navigation and does not require a remote API URL", async () => {
    const fetchMock = vi.fn(async (input: string | URL | Request) => {
      const path = String(input);
      if (path.endsWith("/health")) {
        return new Response(JSON.stringify({ status: "ok", version: "0.2.0" }), { status: 200 });
      }
      return new Response(JSON.stringify({ items: [], page: 1, pageSize: 10, totalItems: 0 }), { status: 200 });
    });
    vi.stubGlobal("fetch", fetchMock);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const router = createMemoryRouter([
      {
        element: <DashboardLayout />,
        children: [{ index: true, element: <DashboardPage /> }],
      },
    ]);

    render(
      <QueryClientProvider client={client}>
        <ThemeController />
        <RouterProvider router={router} />
      </QueryClientProvider>,
    );

    expect(screen.getByRole("heading", { name: "تحلیل‌گر ریسک کد" })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "شروع تحلیل" }).length).toBeGreaterThan(0);
    expect(await screen.findByText(/سرویس محلی متصل است/i)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/health", expect.any(Object));

    const darkModeButton = screen.getByRole("button", { name: "فعال‌کردن حالت تاریک" });
    fireEvent.click(darkModeButton);
    expect(useUIStore.getState().theme).toBe("dark");
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");
    expect(screen.getByRole("button", { name: "فعال‌کردن حالت روشن" })).toBeInTheDocument();
  });
});
