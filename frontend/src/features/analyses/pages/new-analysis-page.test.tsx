import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { NewAnalysisPage } from "@/features/analyses/pages/new-analysis-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("NewAnalysisPage", () => {
  it("validates a local path with the backend before it starts live analysis", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
      if (String(input).endsWith("/analyses/validate-path")) {
        return new Response(JSON.stringify({ valid: true }), { status: 200 });
      }
      return new Response(
        JSON.stringify({
          analysis: { id: "analysis_123" },
        }),
        { status: 202 },
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    const router = createMemoryRouter(
      [
        { path: "/analyses/new", element: <NewAnalysisPage /> },
        { path: "/analyses/:analysisId", element: <p>صفحهٔ پیشرفت تحلیل</p> },
      ],
      { initialEntries: ["/analyses/new"] },
    );
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });

    render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    );

    fireEvent.change(screen.getByLabelText("مسیر مطلق پروژه"), { target: { value: "/workspace/project" } });
    fireEvent.click(screen.getByRole("button", { name: "اعتبارسنجی و شروع تحلیل" }));

    expect(await screen.findByText("صفحهٔ پیشرفت تحلیل")).toBeInTheDocument();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/analyses/validate-path");
    const validationInit = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(JSON.parse(String(validationInit?.body))).toEqual({ projectPath: "/workspace/project" });
    expect(fetchMock.mock.calls[1]?.[0]).toBe("/api/v1/analyses");
    const startInit = fetchMock.mock.calls[1]?.[1] as RequestInit | undefined;
    expect(JSON.parse(String(startInit?.body))).toMatchObject({
      projectPath: "/workspace/project",
      profile: { version: "risk-v1" },
    });
  });

  it("keeps invalid non-local repository URLs out of the API", () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const router = createMemoryRouter([{ path: "/analyses/new", element: <NewAnalysisPage /> }], { initialEntries: ["/analyses/new"] });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });

    render(<QueryClientProvider client={queryClient}><RouterProvider router={router} /></QueryClientProvider>);
    fireEvent.change(screen.getByLabelText("مسیر مطلق پروژه"), { target: { value: "https://github.com/example/project.git" } });
    fireEvent.click(screen.getByRole("button", { name: "اعتبارسنجی و شروع تحلیل" }));

    expect(screen.getByRole("alert")).toHaveTextContent("فقط مسیر محلی پروژه را می‌پذیرد");
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
