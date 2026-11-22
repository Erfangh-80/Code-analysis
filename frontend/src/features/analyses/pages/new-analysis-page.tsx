import { ArrowLeft, Check, FolderSearch, GitCommitHorizontal, LoaderCircle, ShieldCheck, SlidersHorizontal } from "lucide-react";
import { FormEvent, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ApiErrorState } from "@/components/molecules/async-state";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useStartAnalysisMutation, useValidateProjectPathMutation } from "@/hooks/use-analysis-queries";
import { fromDateTimeLocal } from "@/lib/analysis";
import type { RiskMetric } from "@/types/api";

const defaultWeights: Record<RiskMetric, number> = {
  lines_of_code: 0.15,
  max_function_length: 0.25,
  max_class_length: 0.1,
  git_churn: 0.25,
  change_frequency: 0.25,
};

const weightFields: Array<{ metric: RiskMetric; label: string; hint: string }> = [
  { metric: "lines_of_code", label: "خطوط کد", hint: "اندازهٔ فایل" },
  { metric: "max_function_length", label: "طول تابع", hint: "بلندترین تابع" },
  { metric: "max_class_length", label: "طول کلاس", hint: "بلندترین کلاس" },
  { metric: "git_churn", label: "Git Churn", hint: "خطوط تغییرکرده" },
  { metric: "change_frequency", label: "تعداد تغییرات", hint: "commitهای مؤثر" },
];

const supportedLanguages = ["Go", "JavaScript", "Python"];

function isAbsoluteLocalPath(value: string) {
  return value.startsWith("/") || /^[a-zA-Z]:[\\/]/.test(value);
}

function localPathError(value: string) {
  if (!value) {
    return "مسیر پروژه را وارد کنید.";
  }
  if (/^(https?:\/\/|ssh:\/\/|git@)/i.test(value)) {
    return "این بخش فقط مسیر محلی پروژه را می‌پذیرد، نه URL مخزن Git.";
  }
  if (!isAbsoluteLocalPath(value)) {
    return "مسیر باید مطلق باشد؛ مانند /workspace/project یا C:\\workspace\\project.";
  }
  return null;
}

export function NewAnalysisPage() {
  const navigate = useNavigate();
  const startAnalysis = useStartAnalysisMutation();
  const validateProjectPath = useValidateProjectPathMutation();
  const [projectPath, setProjectPath] = useState("");
  const [historyFrom, setHistoryFrom] = useState("");
  const [historyTo, setHistoryTo] = useState("");
  const [weights, setWeights] = useState(defaultWeights);
  const [lowMax, setLowMax] = useState(34);
  const [mediumMax, setMediumMax] = useState(64);
  const [formError, setFormError] = useState<string | null>(null);

  const weightTotal = useMemo(() => Object.values(weights).reduce((sum, value) => sum + value, 0), [weights]);

  function setWeight(metric: RiskMetric, value: string) {
    const parsed = Number(value);
    setWeights((current) => ({ ...current, [metric]: Number.isFinite(parsed) ? parsed : 0 }));
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedPath = projectPath.trim();
    const pathError = localPathError(normalizedPath);
    const from = fromDateTimeLocal(historyFrom);
    const to = fromDateTimeLocal(historyTo);

    if (pathError) {
      setFormError(pathError);
      return;
    }
    if (from && to && new Date(from) > new Date(to)) {
      setFormError("ابتدای بازهٔ Git نمی‌تواند پس از انتهای بازه باشد.");
      return;
    }
    if (Object.values(weights).some((value) => value < 0 || value > 1) || Math.abs(weightTotal - 1) > 0.001) {
      setFormError("مجموع وزن‌های Hotspot باید دقیقاً برابر ۱ باشد.");
      return;
    }
    if (!Number.isInteger(lowMax) || !Number.isInteger(mediumMax) || lowMax < 0 || mediumMax > 99 || lowMax >= mediumMax) {
      setFormError("مرز ریسک کم باید از مرز ریسک متوسط کوچک‌تر باشد.");
      return;
    }

    setFormError(null);
    const analysisRequest = {
      projectPath: normalizedPath,
      history: from || to ? { from: from ?? null, to: to ?? null } : undefined,
      profile: {
        version: "risk-v1",
        weights,
        bands: { lowMax, mediumMax },
      },
    };
    validateProjectPath.mutate(normalizedPath, {
      onSuccess: () => {
        startAnalysis.mutate(analysisRequest, {
          onSuccess: (response) => navigate(`/analyses/${encodeURIComponent(response.analysis.id)}`),
        });
      },
    });
  }

  return (
    <div className="mx-auto grid max-w-5xl gap-6 xl:grid-cols-[minmax(0,1fr)_18rem]">
      <Card>
        <CardHeader>
          <CardTitle>شروع تحلیل پروژهٔ محلی</CardTitle>
          <CardDescription>
            مسیر پروژه، بازهٔ تاریخچه و مدل امتیازدهی را مشخص کنید. سرویس محلی مسیر را اعتبارسنجی می‌کند و سپس تحلیل را در صف قرار می‌دهد.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-7" onSubmit={submit} noValidate>
            <section aria-labelledby="project-selection-title">
              <div className="mb-4 flex items-center gap-2">
                <span className="grid size-7 place-items-center rounded-lg bg-secondary text-secondary-foreground"><FolderSearch className="size-4" /></span>
                <h2 id="project-selection-title" className="text-sm font-black">۱. انتخاب Repository محلی</h2>
              </div>
              <label className="grid gap-2 text-sm font-extrabold" htmlFor="project-path">
                مسیر مطلق پروژه
                <Input
                  id="project-path"
                  dir="ltr"
                  autoComplete="off"
                  placeholder="/workspace/project یا C:\\workspace\\project"
                  value={projectPath}
                  disabled={validateProjectPath.isPending || startAnalysis.isPending}
                  onChange={(event) => {
                    setProjectPath(event.target.value);
                    validateProjectPath.reset();
                  }}
                  aria-describedby="project-path-hint"
                  aria-invalid={Boolean((formError && localPathError(projectPath.trim())) || validateProjectPath.isError)}
                  required
                />
              </label>
              <p id="project-path-hint" className="mt-2 text-xs leading-5 text-muted-foreground">
                فقط یک مسیر محلی قابل‌دسترسی برای سرویس Go وارد کنید. URL گیت و آدرس سرویس خارجی پذیرفته نمی‌شود؛ کد پروژه هم هرگز اجرا نمی‌شود.
              </p>
              <div className="mt-4 rounded-lg border bg-muted/35 p-3">
                <p className="text-xs font-extrabold">زبان‌های قابل تحلیل در API فعلی</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {supportedLanguages.map((language) => (
                    <span key={language} className="inline-flex items-center gap-1.5 rounded-full border bg-card px-2.5 py-1 text-xs font-bold">
                      <Check className="size-3.5 text-success" />
                      {language}
                    </span>
                  ))}
                </div>
                <p className="mt-2 text-xs leading-5 text-muted-foreground">
                  API نسخهٔ فعلی همهٔ زبان‌های پشتیبانی‌شده را به‌صورت خودکار کشف می‌کند و هنوز فیلتر زبانِ قابل ارسال ندارد؛ بنابراین هیچ انتخابی به‌شکل گمراه‌کننده نادیده گرفته نمی‌شود.
                </p>
              </div>
            </section>

            <section className="border-t pt-6" aria-labelledby="history-title">
              <div className="mb-4 flex items-center gap-2">
                <span className="grid size-7 place-items-center rounded-lg bg-secondary text-secondary-foreground"><GitCommitHorizontal className="size-4" /></span>
                <h2 id="history-title" className="text-sm font-black">۲. بازهٔ تاریخچهٔ Git</h2>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="grid gap-2 text-sm font-extrabold" htmlFor="history-from">
                  از تاریخ (اختیاری)
                  <Input id="history-from" type="datetime-local" dir="ltr" value={historyFrom} onChange={(event) => setHistoryFrom(event.target.value)} />
                </label>
                <label className="grid gap-2 text-sm font-extrabold" htmlFor="history-to">
                  تا تاریخ (اختیاری)
                  <Input id="history-to" type="datetime-local" dir="ltr" value={historyTo} onChange={(event) => setHistoryTo(event.target.value)} />
                </label>
              </div>
              <p className="mt-2 text-xs leading-5 text-muted-foreground">اگر بازه خالی باشد، سرویس تمام تاریخچهٔ قابل‌دسترسی Git را بررسی می‌کند. نبودن Git، تحلیل ایستا را متوقف نمی‌کند.</p>
            </section>

            <section className="border-t pt-6" aria-labelledby="profile-title">
              <div className="mb-4 flex items-center gap-2">
                <span className="grid size-7 place-items-center rounded-lg bg-secondary text-secondary-foreground"><SlidersHorizontal className="size-4" /></span>
                <div>
                  <h2 id="profile-title" className="text-sm font-black">۳. تنظیم مدل Hotspot</h2>
                  <p className="mt-1 text-xs text-muted-foreground">مجموع وزن‌ها باید ۱ باشد. در نبود دادهٔ Git، backend وزن‌های در دسترس را بازتنظیم می‌کند.</p>
                </div>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                {weightFields.map((field) => (
                  <label key={field.metric} className="grid gap-1.5 rounded-lg border bg-muted/25 p-3 text-sm font-extrabold" htmlFor={`weight-${field.metric}`}>
                    <span className="flex items-center justify-between gap-2">
                      {field.label}
                      <span className="font-mono text-xs text-primary" dir="ltr">{weights[field.metric].toFixed(2)}</span>
                    </span>
                    <span className="text-xs font-normal text-muted-foreground">{field.hint}</span>
                    <Input
                      id={`weight-${field.metric}`}
                      type="number"
                      min="0"
                      max="1"
                      step="0.05"
                      dir="ltr"
                      value={weights[field.metric]}
                      onChange={(event) => setWeight(field.metric, event.target.value)}
                    />
                  </label>
                ))}
                <div className="rounded-lg border bg-secondary/30 p-3 text-sm">
                  <p className="font-extrabold">مجموع وزن‌ها</p>
                  <p className={`mt-1 font-mono text-xl font-black ${Math.abs(weightTotal - 1) <= 0.001 ? "text-success" : "text-destructive"}`} dir="ltr">
                    {weightTotal.toFixed(2)}
                  </p>
                </div>
              </div>
              <div className="mt-4 grid gap-4 sm:grid-cols-2">
                <label className="grid gap-2 text-sm font-extrabold" htmlFor="low-max">
                  بیشترین امتیاز ریسک کم
                  <Input id="low-max" type="number" min="0" max="98" dir="ltr" value={lowMax} onChange={(event) => setLowMax(Number(event.target.value))} />
                </label>
                <label className="grid gap-2 text-sm font-extrabold" htmlFor="medium-max">
                  بیشترین امتیاز ریسک متوسط
                  <Input id="medium-max" type="number" min="1" max="99" dir="ltr" value={mediumMax} onChange={(event) => setMediumMax(Number(event.target.value))} />
                </label>
              </div>
            </section>

            {formError && <div className="rounded-lg border border-destructive/25 bg-destructive/7 p-3 text-sm text-destructive" role="alert">{formError}</div>}
            {validateProjectPath.isError && <ApiErrorState error={validateProjectPath.error} className="min-h-0" title="مسیر پروژه برای سرویس محلی پذیرفته نشد" />}
            {startAnalysis.isError && <ApiErrorState error={startAnalysis.error} className="min-h-0" title="شروع تحلیل پذیرفته نشد" />}

            <div className="flex flex-wrap gap-3 border-t pt-6">
              <Button type="submit" disabled={validateProjectPath.isPending || startAnalysis.isPending}>
                {validateProjectPath.isPending || startAnalysis.isPending ? <LoaderCircle className="size-4 animate-spin" /> : <FolderSearch className="size-4" />}
                {validateProjectPath.isPending ? "در حال بررسی مسیر…" : startAnalysis.isPending ? "در حال شروع تحلیل…" : "اعتبارسنجی و شروع تحلیل"}
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
            <li>مسیر ابتدا توسط backend اعتبارسنجی می‌شود و اطلاعات مطلق آن در گزارش‌ها برنمی‌گردد.</li>
            <li>پس از پذیرش، پیشرفت مرحله‌ها به‌صورت زنده در صفحهٔ تحلیل نمایش داده می‌شود.</li>
            <li>خطای مسیر نامعتبر، مسیر خارج از دسترس یا تنظیمات نادرست به‌صورت شفاف نمایش داده می‌شود.</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
