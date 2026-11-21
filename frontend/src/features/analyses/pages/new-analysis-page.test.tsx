import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { NewAnalysisPage } from "@/features/analyses/pages/new-analysis-page";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("NewAnalysisPage", () => {
  it("starts a valid local analysis and takes the user to live progress", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) =>
      new Response(
        JSON.stringify({
          analysis: { id: "analysis_123" },
        }),
        { status: 202 },
      ),
    );
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
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(JSON.parse(String(init?.body))).toMatchObject({
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
