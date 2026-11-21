import { create } from "zustand";
import { persist } from "zustand/middleware";

interface UIState {
  isMobileSidebarOpen: boolean;
  density: "comfortable" | "compact";
  openMobileSidebar: () => void;
  closeMobileSidebar: () => void;
  toggleMobileSidebar: () => void;
  setDensity: (density: UIState["density"]) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      isMobileSidebarOpen: false,
      density: "comfortable",
      openMobileSidebar: () => set({ isMobileSidebarOpen: true }),
      closeMobileSidebar: () => set({ isMobileSidebarOpen: false }),
      toggleMobileSidebar: () =>
        set((state) => ({ isMobileSidebarOpen: !state.isMobileSidebarOpen })),
      setDensity: (density) => set({ density }),
    }),
    {
      name: "code-risk-analyzer-ui",
      partialize: (state) => ({ density: state.density }),
    },
  ),
);
