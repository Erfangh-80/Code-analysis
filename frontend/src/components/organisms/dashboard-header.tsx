import { Menu, Moon, ShieldCheck, Sun } from "lucide-react";

import { APIConnectionStatus } from "@/components/molecules/api-connection-status";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useUIStore } from "@/store/ui-store";

interface DashboardHeaderProps {
  title: string;
  description: string;
}

export function DashboardHeader({ title, description }: DashboardHeaderProps) {
  const openMobileSidebar = useUIStore((state) => state.openMobileSidebar);
  const theme = useUIStore((state) => state.theme);
  const setTheme = useUIStore((state) => state.setTheme);
  const isDark = theme === "dark";

  return (
    <header className="flex flex-col gap-4 border-b bg-background/75 px-5 py-4 backdrop-blur-sm sm:px-7 lg:px-10">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <Button
            variant="outline"
            size="icon"
            className="mt-0.5 lg:hidden"
            aria-label="باز کردن منوی ناوبری"
            onClick={openMobileSidebar}
          >
            <Menu className="size-5" />
          </Button>
          <div>
            <h1 className="text-xl font-black tracking-tight sm:text-2xl">{title}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            aria-label={isDark ? "فعال‌کردن حالت روشن" : "فعال‌کردن حالت تاریک"}
            aria-pressed={isDark}
            title={isDark ? "فعال‌کردن حالت روشن" : "فعال‌کردن حالت تاریک"}
            onClick={() => setTheme(isDark ? "light" : "dark")}
          >
            {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>
          <APIConnectionStatus />
        </div>
      </div>

      <Dialog>
        <DialogTrigger asChild>
          <Button variant="ghost" size="sm" className="w-fit text-muted-foreground">
            <ShieldCheck className="size-4" />
            حریم داده‌های محلی
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>مرز امنیتی تحلیل محلی</DialogTitle>
            <DialogDescription>
              مسیر پروژه فقط برای سرویس Go روی همین دستگاه ارسال می‌شود. رابط کاربری از مسیرهای نسبی API استفاده می‌کند و
              مستقیماً به سرویس خارجی یا مخزن شما وصل نمی‌شود.
            </DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    </header>
  );
}
