import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, FolderSearch, LoaderCircle, ShieldCheck } from "lucide-react";
import { FormEvent, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { analysisApi } from "@/services/api/analysis-api";

export function NewAnalysisPage() {
  const queryClient = useQueryClient();
  const [projectPath, setProjectPath] = useState("");

  const startAnalysis = useMutation({
    mutationFn: analysisApi.start,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["analyses"] });
    },
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedPath = projectPath.trim();
    if (!normalizedPath) {
      return;
    }
    startAnalysis.mutate(normalizedPath);
  }

  return (
    <div className="mx-auto grid max-w-4xl gap-6 lg:grid-cols-[minmax(0,1fr)_17rem]">
      <Card>
        <CardHeader>
          <CardTitle>شروع تحلیل پروژهٔ محلی</CardTitle>
          <CardDescription>
            فقط یک مسیر مطلق روی دستگاهی که سرویس Go اجرا می‌شود وارد کنید. URL گیت یا آدرس سرویس خارجی پذیرفته نمی‌شود.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-5" onSubmit={submit} noValidate>
            <label className="grid gap-2 text-sm font-extrabold" htmlFor="project-path">
              مسیر پروژه
              <Input
                id="project-path"
                dir="ltr"
                autoComplete="off"
                placeholder="/Users/name/workspace/project"
                value={projectPath}
                onChange={(event) => setProjectPath(event.target.value)}
                aria-describedby="project-path-hint"
                required
              />
            </label>
            <p id="project-path-hint" className="text-xs leading-5 text-muted-foreground">
              تحلیل‌گر ابتدا فایل‌های پشتیبانی‌شده را پیدا می‌کند، سپس تحلیل ایستا و در صورت وجود، تاریخچهٔ Git را بدون اجرای
              کد پروژه بررسی می‌کند.
            </p>

            {startAnalysis.isError && (
              <div className="rounded-lg border border-destructive/25 bg-destructive/8 p-3 text-sm text-destructive" role="alert">
                {startAnalysis.error.message}
              </div>
            )}

            {startAnalysis.isSuccess && (
              <div className="rounded-lg border border-success/25 bg-success/10 p-3 text-sm text-[color:var(--success)]" role="status">
                تحلیل با شناسهٔ <span dir="ltr" className="font-mono text-xs">{startAnalysis.data.analysis.id}</span> در صف قرار گرفت.
                <Link className="mr-2 font-extrabold underline" to="/analyses">
                  مشاهده وضعیت
                </Link>
              </div>
            )}

            <div className="flex flex-wrap gap-3">
              <Button type="submit" disabled={startAnalysis.isPending || projectPath.trim() === ""}>
                {startAnalysis.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <FolderSearch className="size-4" />}
                {startAnalysis.isPending ? "در حال ثبت…" : "ثبت و شروع تحلیل"}
              </Button>
              <Button type="button" variant="outline" asChild>
                <Link to="/analyses">
                  بازگشت به تحلیل‌ها
                  <ArrowLeft className="size-4" />
                </Link>
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="h-fit bg-secondary/35">
        <CardHeader>
          <ShieldCheck className="size-6 text-primary" aria-hidden="true" />
          <CardTitle className="mt-2">تحلیل قابل‌اعتماد</CardTitle>
        </CardHeader>
        <CardContent className="text-sm leading-6 text-muted-foreground">
          <ul className="space-y-3">
            <li>کد پروژه build، import یا اجرا نمی‌شود.</li>
            <li>نبودن تاریخچهٔ Git باعث توقف تحلیل ایستا نمی‌شود.</li>
            <li>مسیرهای مطلق در پاسخ‌ها و گزارش‌های قابل‌نمایش افشا نمی‌شوند.</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
