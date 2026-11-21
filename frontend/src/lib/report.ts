import type { AnalysisFileResult, AnalysisReport, RiskLevel } from "@/types/api";

export function fileResultsFromReport(report: AnalysisReport): AnalysisFileResult[] {
  const staticFiles = report.artifacts.static?.files ?? [];
  const gitByPath = new Map(report.artifacts.git?.files.map((file) => [file.path, file]) ?? []);
  const riskByPath = new Map(report.artifacts.risk?.files.map((file) => [file.path, file]) ?? []);

  return staticFiles.map((analysis) => ({
    analysis,
    git: gitByPath.get(analysis.path),
    risk: riskByPath.get(analysis.path),
  }));
}

export function riskMatches(levels: RiskLevel[], level: RiskLevel | undefined) {
  return levels.length === 0 || (level !== undefined && levels.includes(level));
}
