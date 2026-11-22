import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { NewAnalysisPage } from "@/features/analyses/pages/new-analysis-page";

function renderNewAnalysisPage() {
  const router = createMemoryRouter(
    [
      { path: "/analyses/new", element: <NewAnalysisPage /> },
      { path: "/analyses/:analysisId", element: <p>صفحهٔ پیشرفت تحلیل</p> },
    ],
    { initialEntries: ["/analyses/new"] },
  );
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("analysis start integration", () => {
  it("does not start analysis when backend path validation rejects the project", async () => {
    const fetchMock = vi.fn(
      async (_input: RequestInfo | URL, _init?: RequestInit) =>
        new Response(
          JSON.stringify({
            error: {
              code: "VALIDATION_ERROR",
              message: "مسیر پروژه در محدودهٔ مجاز نیست.",
            },
          }),
          { status: 400 },
        ),
    );
    vi.stubGlobal("fetch", fetchMock);
    renderNewAnalysisPage();

    fireEvent.change(screen.getByLabelText("مسیر مطلق پروژه"), {
      target: { value: "/outside/allowed-root" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "اعتبارسنجی و شروع تحلیل" }),
    );

    expect(
      await screen.findByText("مسیر پروژه در محدودهٔ مجاز نیست."),
    ).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/analyses/validate-path");
  });

  it("shows an actionable local-service connection error when the backend is unavailable", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("network unavailable");
      }),
    );
    renderNewAnalysisPage();

    fireEvent.change(screen.getByLabelText("مسیر مطلق پروژه"), {
      target: { value: "/workspace/project" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "اعتبارسنجی و شروع تحلیل" }),
    );

    expect(
      await screen.findByText(
        "ارتباط با سرویس محلی برقرار نشد. اجرا بودن سرویس را بررسی کنید.",
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByText("صفحهٔ پیشرفت تحلیل")).not.toBeInTheDocument(),
    );
  });
});
