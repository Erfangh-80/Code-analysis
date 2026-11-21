import type { AnalysisRunStatus, AnalysisStage, FileAnalysisStatus, RiskLevel, RiskMetric, StageStatus, SupportedLanguage } from "@/types/api";

export const riskLevelLabels: Record<RiskLevel, string> = {
  low: "کم",
  medium: "متوسط",
  high: "زیاد",
};

export const languageLabels: Record<SupportedLanguage, string> = {
  go: "Go",
  javascript: "JavaScript",
  python: "Python",
  unknown: "نامشخص",
};

export const metricLabels: Record<RiskMetric, string> = {
  lines_of_code: "خطوط کد",
  max_function_length: "بیشترین طول تابع",
  max_class_length: "بیشترین طول کلاس",
  git_churn: "Git Churn",
  change_frequency: "تعداد تغییرات",
};

export const stageLabels: Record<AnalysisStage, string> = {
  validation: "اعتبارسنجی",
  discovery: "شناسایی فایل‌ها",
  static_analysis: "تحلیل ایستا",
  git_analysis: "تحلیل تاریخچهٔ Git",
  risk_scoring: "محاسبهٔ ریسک",
};

export const stageStatusLabels: Record<StageStatus, string> = {
  pending: "در انتظار",
  running: "در حال اجرا",
  completed: "تکمیل‌شده",
  unavailable: "در دسترس نیست",
  failed: "ناموفق",
  cancelled: "لغوشده",
  skipped: "رد شده",
};

export const fileStatusLabels: Record<FileAnalysisStatus, string> = {
  analyzed: "تحلیل‌شده",
  empty: "خالی",
  no_executable_code: "بدون کد اجرایی",
  fallback: "تحلیل جایگزین",
  parse_error: "خطای parse",
  unsupported: "پشتیبانی‌نشده",
  read_error: "خطای خواندن",
};

export function isTerminalAnalysisStatus(status: AnalysisRunStatus) {
  return status === "completed" || status === "failed" || status === "cancelled";
}

export function formatNumber(value: number | null | undefined) {
  if (value === null || value === undefined) {
    return "—";
  }
  return new Intl.NumberFormat("fa-IR").format(value);
}

export function toDateTimeLocal(value?: string | null) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

export function fromDateTimeLocal(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}
