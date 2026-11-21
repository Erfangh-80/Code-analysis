import { ArrowLeft, FileWarning } from "lucide-react";
import { Link } from "react-router-dom";

import { EmptyState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { RiskLevelBadge } from "@/features/analyses/components/risk-level-badge";
import { formatNumber } from "@/lib/analysis";
import type { Hotspot } from "@/types/api";

interface TopRiskyFilesProps {
  analysisId: string;
  files: Hotspot[];
}

export function TopRiskyFiles({ analysisId, files }: TopRiskyFilesProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle>پرریسک‌ترین فایل‌ها</CardTitle>
          <CardDescription>فایل‌ها بر اساس Hotspot Score مرتب شده‌اند.</CardDescription>
        </div>
        <FileWarning className="size-5 text-destructive" aria-hidden="true" />
      </CardHeader>
      <CardContent>
        {files.length === 0 ? (
          <EmptyState title="Hotspotی ثبت نشده است" description="برای این تحلیل دادهٔ ریسک قابل نمایش وجود ندارد." className="min-h-48" />
        ) : (
          <ol className="divide-y">
            {files.map((file) => (
              <li key={file.path} className="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
                <div className="min-w-0">
                  <Link className="block truncate font-mono text-xs font-bold hover:text-primary hover:underline" dir="ltr" to={`/analyses/${encodeURIComponent(analysisId)}/files/detail?path=${encodeURIComponent(file.path)}`}>
                    {file.path}
                  </Link>
                  <p className="mt-1 text-xs text-muted-foreground">Churn: {formatNumber(file.metrics.gitChurn)} · تغییرات: {formatNumber(file.metrics.changeFrequency)}</p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <RiskLevelBadge level={file.level} />
                  <span className="grid size-9 place-items-center rounded-lg bg-destructive/10 text-sm font-black text-destructive">{file.score}</span>
                </div>
              </li>
            ))}
          </ol>
        )}
        {files.length > 0 && (
          <Button className="mt-5 w-full" variant="outline" size="sm" asChild>
            <Link to={`/analyses/${encodeURIComponent(analysisId)}/files`}>
              مشاهدهٔ همهٔ فایل‌ها
              <ArrowLeft className="size-3.5" />
            </Link>
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
