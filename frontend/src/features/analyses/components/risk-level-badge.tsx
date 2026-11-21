import { Badge, type badgeVariants } from "@/components/ui/badge";
import { riskLevelLabels } from "@/lib/analysis";
import type { RiskLevel } from "@/types/api";

type BadgeVariant = NonNullable<Parameters<typeof badgeVariants>[0]>["variant"];

const variants: Record<RiskLevel, BadgeVariant> = {
  low: "success",
  medium: "warning",
  high: "destructive",
};

export function RiskLevelBadge({ level }: { level: RiskLevel }) {
  return <Badge variant={variants[level]}>ریسک {riskLevelLabels[level]}</Badge>;
}
