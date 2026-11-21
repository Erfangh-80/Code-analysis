import { Check, Palette } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useUIStore } from "@/store/ui-store";

export function SettingsPage() {
  const density = useUIStore((state) => state.density);
  const setDensity = useUIStore((state) => state.setDensity);

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="grid size-10 place-items-center rounded-xl bg-secondary text-secondary-foreground">
              <Palette className="size-5" />
            </span>
            <div>
              <CardTitle>تنظیمات رابط کاربری</CardTitle>
              <CardDescription>این ترجیحات فقط در مرورگر شما نگهداری می‌شوند.</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="grid gap-5 border-t pt-5 sm:grid-cols-2 sm:items-end">
          <label className="grid gap-2 text-sm font-extrabold">
            تراکم نمایش فهرست‌ها
            <Select value={density} onValueChange={(value) => setDensity(value as "comfortable" | "compact")}>
              <SelectTrigger aria-label="تراکم نمایش فهرست‌ها">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="comfortable">راحت</SelectItem>
                <SelectItem value="compact">فشرده</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <p className="flex gap-2 text-sm leading-6 text-muted-foreground">
            <Check className="mt-0.5 size-4 shrink-0 text-success" />
            انتخاب شما با Zustand در state رابط ذخیره می‌شود و به API تحلیل ارسال نمی‌شود.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
