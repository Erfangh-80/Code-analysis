import { Braces } from "lucide-react";

import { cn } from "@/lib/utils";

interface BrandMarkProps {
  compact?: boolean;
  className?: string;
}

export function BrandMark({ compact = false, className }: BrandMarkProps) {
  return (
    <div className={cn("flex items-center gap-3", className)}>
      <span className="grid size-10 place-items-center rounded-xl bg-primary text-primary-foreground shadow-sm">
        <Braces className="size-5" aria-hidden="true" />
      </span>
      {!compact && (
        <span className="grid leading-tight">
          <strong className="text-sm font-black text-foreground">تحلیل‌گر ریسک کد</strong>
          <span className="text-xs text-muted-foreground">Code Risk Analyzer</span>
        </span>
      )}
    </div>
  );
}
