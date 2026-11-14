import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import HomePage from "./page";

describe("HomePage", () => {
  it("explains the local-only analysis boundary", () => {
    render(<HomePage />);

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: /understand repository risk locally/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/no source code is uploaded/i),
    ).toBeInTheDocument();
  });

  it("does not present unimplemented analysis results as real data", () => {
    render(<HomePage />);

    expect(
      screen.getByText(/analysis workspace is coming next/i),
    ).toBeInTheDocument();
    expect(screen.queryByText(/critical files/i)).not.toBeInTheDocument();
  });
});
