import { Outlet, useMatches } from "react-router-dom";

import { AppSidebar } from "@/components/organisms/app-sidebar";
import { DashboardHeader } from "@/components/organisms/dashboard-header";
import { useUIStore } from "@/store/ui-store";

interface RouteHandle {
  title?: string;
  description?: string;
}

const defaultHeader: Required<RouteHandle> = {
  title: "تحلیل‌گر ریسک کد",
  description: "فضای کاری محلی برای شناخت کیفیت و ریسک تغییرات پروژه.",
};

export function DashboardLayout() {
  const matches = useMatches();
  const density = useUIStore((state) => state.density);
  const matchWithHeader = [...matches].reverse().find((match) => match.handle) as
    | (typeof matches)[number] & { handle?: RouteHandle }
    | undefined;
  const header = { ...defaultHeader, ...matchWithHeader?.handle };

  return (
    <div className="min-h-screen bg-background text-foreground" dir="rtl" data-density={density}>
      <AppSidebar />
      <div className="min-h-screen lg:mr-72">
        <DashboardHeader title={header.title} description={header.description} />
        <main className="page-enter mx-auto w-full max-w-7xl p-5 sm:p-7 lg:p-10">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
