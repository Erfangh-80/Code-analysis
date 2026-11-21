import { Badge, type badgeVariants } from "@/components/ui/badge";
import type { AnalysisRunStatus } from "@/types/api";

type BadgeVariant = NonNullable<Parameters<typeof badgeVariants>[0]>["variant"];

const statusConfig: Record<AnalysisRunStatus, { label: string; variant: BadgeVariant }> = {
  queued: { label: "در صف", variant: "muted" },
  running: { label: "در حال اجرا", variant: "default" },
  completed: { label: "تکمیل‌شده", variant: "success" },
  failed: { label: "ناموفق", variant: "destructive" },
  cancelled: { label: "لغو شده", variant: "warning" },
};

export function AnalysisStatusBadge({ status }: { status: AnalysisRunStatus }) {
  const config = statusConfig[status];
  return <Badge variant={config.variant}>{config.label}</Badge>;
}
