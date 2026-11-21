import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Theme = "light" | "dark";

interface UIState {
  isMobileSidebarOpen: boolean;
  density: "comfortable" | "compact";
  theme: Theme;
  openMobileSidebar: () => void;
  closeMobileSidebar: () => void;
  toggleMobileSidebar: () => void;
  setDensity: (density: UIState["density"]) => void;
  setTheme: (theme: Theme) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      isMobileSidebarOpen: false,
      density: "comfortable",
      theme: "light",
      openMobileSidebar: () => set({ isMobileSidebarOpen: true }),
      closeMobileSidebar: () => set({ isMobileSidebarOpen: false }),
      toggleMobileSidebar: () =>
        set((state) => ({ isMobileSidebarOpen: !state.isMobileSidebarOpen })),
      setDensity: (density) => set({ density }),
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: "code-risk-analyzer-ui",
      partialize: (state) => ({ density: state.density, theme: state.theme }),
    },
  ),
);
