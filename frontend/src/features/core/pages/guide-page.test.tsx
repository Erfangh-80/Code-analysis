import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GuidePage } from "@/features/core/pages/guide-page";

describe("GuidePage", () => {
  it("explains the metrics and risk levels without external data", () => {
    render(<GuidePage />);

    expect(screen.getByText("Code Risk Analyzer چه کار می‌کند؟")).toBeInTheDocument();
    expect(screen.getByText("حجم تغییرات (Git Churn)")).toBeInTheDocument();
    expect(screen.getByText("زیاد (۶۵ تا ۱۰۰)")).toBeInTheDocument();
    expect(screen.getByText(/سیگنال اولویت‌بندی/)).toBeInTheDocument();
  });
});
