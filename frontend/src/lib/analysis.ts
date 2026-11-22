import type { BadgeVariant } from "@/components/ui/badge";
import type { FileAnalysisStatus, RiskLevel, RiskMetric, SupportedLanguage } from "@/types/api";

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
  git_churn: "حجم تغییرات (Churn)",
  change_frequency: "دفعات تغییر",
};

/** Badge colors keep status/language/risk readable without relying on color alone. */
export const languageBadgeVariants: Record<SupportedLanguage, BadgeVariant> = {
  go: "info",
  javascript: "warning",
  python: "success",
  unknown: "muted",
};

export const statusBadgeVariants: Record<FileAnalysisStatus, BadgeVariant> = {
  analyzed: "success",
  empty: "muted",
  no_executable_code: "muted",
  fallback: "warning",
  parse_error: "destructive",
  unsupported: "warning",
  read_error: "destructive",
};

export const riskLevelBadgeVariants: Record<RiskLevel, BadgeVariant> = {
  low: "success",
  medium: "warning",
  high: "destructive",
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

export function formatNumber(value: number | null | undefined) {
  if (value === null || value === undefined) {
    return "—";
  }
  return new Intl.NumberFormat("fa-IR").format(value);
}

export function fromDateTimeLocal(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}
