import { Cell, Legend, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

import { EmptyState } from "@/components/molecules/async-state";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { RiskLevelCounts } from "@/types/api";

interface RiskDistributionChartProps {
  counts: RiskLevelCounts;
}

const riskSlices = [
  { key: "high", label: "ریسک زیاد", color: "var(--destructive)" },
  { key: "medium", label: "ریسک متوسط", color: "var(--warning)" },
  { key: "low", label: "ریسک کم", color: "var(--success)" },
] as const;

export function RiskDistributionChart({ counts }: RiskDistributionChartProps) {
  const data = riskSlices.map((slice) => ({ ...slice, value: counts[slice.key] }));
  const total = data.reduce((sum, item) => sum + item.value, 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle>توزیع ریسک فایل‌ها</CardTitle>
        <CardDescription>سطح ریسک از گزارش تکمیل‌شدهٔ backend خوانده می‌شود.</CardDescription>
      </CardHeader>
      <CardContent>
        {total === 0 ? (
          <EmptyState title="فایل امتیازدهی‌شده‌ای وجود ندارد" description="ممکن است تحلیل بدون دادهٔ قابل امتیازدهی تکمیل شده باشد." className="min-h-56" />
        ) : (
          <div className="h-64" dir="ltr">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={data} dataKey="value" nameKey="label" cx="50%" cy="46%" innerRadius={53} outerRadius={82} paddingAngle={3} stroke="var(--card)">
                  {data.map((slice) => <Cell key={slice.key} fill={slice.color} />)}
                </Pie>
                <Tooltip
                  formatter={(value, name) => [new Intl.NumberFormat("fa-IR").format(Number(value)), name]}
                  contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: "0.75rem", color: "var(--foreground)", direction: "rtl" }}
                />
                <Legend verticalAlign="bottom" iconType="circle" wrapperStyle={{ direction: "rtl", fontSize: "0.75rem" }} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
