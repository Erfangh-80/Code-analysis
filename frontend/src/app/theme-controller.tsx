import { useEffect } from "react";

import { useUIStore } from "@/store/ui-store";

const themeColors = {
  light: "#0f766e",
  dark: "#10282d",
} as const;

// Keeps document-level tokens in sync so portals, native controls, and browser
// chrome use the selected appearance as well as the dashboard shell.
export function ThemeController() {
  const theme = useUIStore((state) => state.theme);

  useEffect(() => {
    const root = document.documentElement;
    root.dataset.theme = theme;
    root.style.colorScheme = theme;
    document.querySelector('meta[name="theme-color"]')?.setAttribute("content", themeColors[theme]);
  }, [theme]);

  return null;
}
