import { Info } from "lucide-react";
import { Cell, CartesianGrid, ResponsiveContainer, Scatter, ScatterChart, Tooltip, XAxis, YAxis, ZAxis } from "recharts";
import { useNavigate } from "react-router-dom";

import { EmptyState } from "@/components/molecules/async-state";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { metricLabels, riskLevelLabels } from "@/lib/analysis";
import type { HeatmapCell, HeatmapData, RiskLevel } from "@/types/api";

interface RiskHeatmapProps {
  analysisId: string;
  data: HeatmapData;
}

const levelColor: Record<RiskLevel, string> = {
  low: "var(--success)",
  medium: "var(--warning)",
  high: "var(--destructive)",
};

function HeatmapTooltip({ active, payload }: { active?: boolean; payload?: ReadonlyArray<{ payload?: HeatmapCell }> }) {
  const cell = payload?.[0]?.payload;
  if (!active || !cell) {
    return null;
  }

  return (
    <div className="max-w-64 rounded-xl border bg-popover p-3 text-right text-xs shadow-float" dir="rtl">
      <p className="truncate font-mono font-bold" dir="ltr">{cell.path}</p>
      <p className="mt-1 text-muted-foreground">ماژول: <span dir="ltr">{cell.module}</span></p>
      <div className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 border-t pt-2">
        <span>امتیاز: <strong>{cell.score}</strong></span>
        <span>ریسک: <strong>{riskLevelLabels[cell.level]}</strong></span>
        <span>مقدار محور افقی: <strong>{cell.xValue}</strong></span>
        <span>مقدار محور عمودی: <strong>{cell.yValue}</strong></span>
      </div>
      <p className="mt-2 text-[0.7rem] text-muted-foreground">برای دیدن جزئیات فایل کلیک کنید.</p>
    </div>
  );
}

export function RiskHeatmap({ analysisId, data }: RiskHeatmapProps) {
  const navigate = useNavigate();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Heatmap ریسک فایل‌ها</CardTitle>
        <CardDescription>
          پراکندگی دوبعدی <span className="font-extrabold">{metricLabels[data.xMetric]}</span> × <span className="font-extrabold">{metricLabels[data.yMetric]}</span>؛ اندازهٔ نقطه از Hotspot Score می‌آید.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {data.cells.length === 0 ? (
          <EmptyState title="داده‌ای برای Heatmap موجود نیست" description="این تحلیل دادهٔ ریسک کافی برای رسم نقاط ندارد." className="min-h-72" />
        ) : (
          <>
            <div className="h-[22.5rem] min-h-72" dir="ltr">
              <ResponsiveContainer width="100%" height="100%">
                <ScatterChart margin={{ top: 16, right: 16, bottom: 20, left: 8 }}>
                  <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" />
                  <XAxis
                    type="number"
                    dataKey="xValue"
                    name={metricLabels[data.xMetric]}
                    tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
                    label={{ value: metricLabels[data.xMetric], position: "insideBottom", offset: -10, fill: "var(--muted-foreground)", fontSize: 11 }}
                  />
                  <YAxis
                    type="number"
                    dataKey="yValue"
                    name={metricLabels[data.yMetric]}
                    tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
                    label={{ value: metricLabels[data.yMetric], angle: -90, position: "insideLeft", fill: "var(--muted-foreground)", fontSize: 11 }}
                  />
                  <ZAxis type="number" dataKey="score" range={[55, 260]} name="امتیاز ریسک" />
                  <Tooltip
                    cursor={{ stroke: "var(--ring)", strokeDasharray: "4 4" }}
                    content={(props) => <HeatmapTooltip active={props.active} payload={props.payload as ReadonlyArray<{ payload?: HeatmapCell }> | undefined} />}
                  />
                  <Scatter
                    data={data.cells}
                    name="فایل‌ها"
                    onClick={(point) => {
                      const cell = (point as { payload?: HeatmapCell }).payload;
                      if (cell) {
                        navigate(`/analyses/${encodeURIComponent(analysisId)}/files/detail?path=${encodeURIComponent(cell.path)}`);
                      }
                    }}
                    className="cursor-pointer"
                  >
                    {data.cells.map((cell) => <Cell key={cell.path} fill={levelColor[cell.level]} stroke="var(--card)" strokeWidth={2} />)}
                  </Scatter>
                </ScatterChart>
              </ResponsiveContainer>
            </div>
            <div className="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 border-t pt-4 text-xs">
              {(["low", "medium", "high"] as RiskLevel[]).map((level) => (
                <span key={level} className="inline-flex items-center gap-2">
                  <i className="size-2.5 rounded-full" style={{ backgroundColor: levelColor[level] }} />
                  ریسک {riskLevelLabels[level]}
                </span>
              ))}
              <span className="mr-auto inline-flex items-center gap-1.5 text-muted-foreground"><Info className="size-3.5" />اندازهٔ نقطه = امتیاز ریسک</span>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
