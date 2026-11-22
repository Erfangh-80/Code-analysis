import { BookOpen, ScanSearch, X } from "lucide-react";

import { BrandMark } from "@/components/atoms/brand-mark";
import { NavigationItem } from "@/components/molecules/navigation-item";
import { Button } from "@/components/ui/button";
import { useUIStore } from "@/store/ui-store";

const navigation = [
  { to: "/", icon: ScanSearch, label: "هستهٔ تحلیل", end: true },
  { to: "/guide", icon: BookOpen, label: "راهنما" },
];

export function AppSidebar() {
  const isMobileSidebarOpen = useUIStore((state) => state.isMobileSidebarOpen);
  const closeMobileSidebar = useUIStore((state) => state.closeMobileSidebar);

  return (
    <>
      {isMobileSidebarOpen && <button aria-label="بستن منوی ناوبری" className="fixed inset-0 z-40 bg-foreground/30 lg:hidden" onClick={closeMobileSidebar} />}
      <aside aria-label="ناوبری اصلی" className={`fixed inset-y-0 right-0 z-50 flex w-72 flex-col border-l bg-card px-4 py-5 shadow-float transition-transform duration-200 lg:translate-x-0 lg:shadow-none ${isMobileSidebarOpen ? "translate-x-0" : "translate-x-full"}`}>
        <div className="flex items-center justify-between px-2">
          <BrandMark />
          <Button variant="ghost" size="icon" className="lg:hidden" aria-label="بستن منوی ناوبری" onClick={closeMobileSidebar}><X className="size-5" /></Button>
        </div>
        <nav className="mt-9 grid gap-1">
          <p className="px-3 pb-2 text-[0.69rem] font-black tracking-[0.12em] text-muted-foreground uppercase">فازهای تحلیلی</p>
          {navigation.map((item) => <NavigationItem key={item.to} {...item} onNavigate={closeMobileSidebar} />)}
        </nav>
        <div className="mt-auto rounded-xl border bg-muted/55 p-4">
          <p className="text-sm font-extrabold">داده‌ها روی دستگاه شما می‌مانند</p>
          <p className="mt-1.5 text-xs leading-5 text-muted-foreground">این خروجی فقط API محلی Go را برای تحلیل ایستا، Git و Hotspot صدا می‌زند و کد پروژه را اجرا نمی‌کند.</p>
        </div>
      </aside>
    </>
  );
}
