import { ArrowLeft, FileWarning, LoaderCircle } from "lucide-react";
import { Link } from "react-router-dom";

import { AnalysisStatusBadge } from "@/features/analyses/components/analysis-status-badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatDateTime } from "@/lib/utils";
import { useAnalysesQuery } from "@/hooks/use-analyses-query";

interface AnalysesTableProps {
  showAll?: boolean;
}

export function AnalysesTable({ showAll = false }: AnalysesTableProps) {
  const analyses = useAnalysesQuery();

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle>{showAll ? "فهرست تحلیل‌ها" : "تحلیل‌های اخیر"}</CardTitle>
          <CardDescription>
            {showAll
              ? "وضعیت تحلیل‌های ذخیره‌شده در سرویس محلی را اینجا ببینید."
              : "این داده‌ها فقط پس از پاسخ سرویس محلی نمایش داده می‌شوند."}
          </CardDescription>
        </div>
        {!showAll && (
          <Button variant="outline" size="sm" asChild>
            <Link to="/analyses">
              مشاهده همه
              <ArrowLeft className="size-3.5" />
            </Link>
          </Button>
        )}
      </CardHeader>
      <CardContent>
        {analyses.isPending && (
          <div className="flex min-h-40 items-center justify-center gap-2 text-sm text-muted-foreground">
            <LoaderCircle className="size-4 animate-spin" />
            در حال دریافت تحلیل‌ها از سرویس محلی…
          </div>
        )}

        {analyses.isError && (
          <div className="flex min-h-40 flex-col items-center justify-center rounded-lg border border-dashed bg-muted/45 p-6 text-center">
            <FileWarning className="size-6 text-warning" aria-hidden="true" />
            <p className="mt-3 text-sm font-extrabold">امکان دریافت تحلیل‌ها وجود ندارد</p>
            <p className="mt-1 max-w-md text-xs leading-5 text-muted-foreground">
              ابتدا سرویس محلی Go را اجرا کنید؛ سپس این صفحه به‌طور خودکار به‌روزرسانی می‌شود.
            </p>
          </div>
        )}

        {analyses.isSuccess && analyses.data.items.length === 0 && (
          <div className="flex min-h-40 flex-col items-center justify-center rounded-lg border border-dashed bg-muted/45 p-6 text-center">
            <p className="text-sm font-extrabold">هنوز تحلیلی ثبت نشده است</p>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">
              برای شروع، مسیر یک پروژهٔ محلی را در بخش «شروع تحلیل» وارد کنید.
            </p>
            <Button className="mt-4" size="sm" asChild>
              <Link to="/analyses/new">شروع تحلیل</Link>
            </Button>
          </div>
        )}

        {analyses.isSuccess && analyses.data.items.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>شناسه تحلیل</TableHead>
                <TableHead>وضعیت</TableHead>
                <TableHead>پیشرفت</TableHead>
                <TableHead>ایجاد شده</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {analyses.data.items.map((analysis) => (
                <TableRow key={analysis.id}>
                  <TableCell className="max-w-48 truncate font-mono text-xs" dir="ltr">
                    {analysis.id}
                  </TableCell>
                  <TableCell>
                    <AnalysisStatusBadge status={analysis.status} />
                  </TableCell>
                  <TableCell>{analysis.progress.percent}٪</TableCell>
                  <TableCell className="whitespace-nowrap text-muted-foreground">
                    {formatDateTime(analysis.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
