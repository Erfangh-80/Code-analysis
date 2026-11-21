import type { LucideIcon } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";

interface OverviewMetricCardProps {
  label: string;
  value: string | number;
  description: string;
  icon: LucideIcon;
  tone?: "primary" | "accent" | "warning";
}

const toneClasses = {
  primary: "bg-primary/10 text-primary",
  accent: "bg-accent text-accent-foreground",
  warning: "bg-warning/15 text-[color:var(--warning)]",
};

export function OverviewMetricCard({
  label,
  value,
  description,
  icon: Icon,
  tone = "primary",
}: OverviewMetricCardProps) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="text-sm font-bold text-muted-foreground">{label}</p>
            <p className="mt-3 text-3xl font-black tracking-tight">{value}</p>
          </div>
          <span className={`grid size-10 place-items-center rounded-xl ${toneClasses[tone]}`}>
            <Icon className="size-5" aria-hidden="true" />
          </span>
        </div>
        <p className="mt-4 text-xs leading-5 text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  );
}
