export type SupportedLanguage = "go" | "javascript" | "python" | "unknown";
export type FileAnalysisStatus =
  | "analyzed"
  | "empty"
  | "no_executable_code"
  | "fallback"
  | "parse_error"
  | "unsupported"
  | "read_error";
export type RiskLevel = "low" | "medium" | "high";
export type RiskMetric =
  | "lines_of_code"
  | "max_function_length"
  | "max_class_length"
  | "git_churn"
  | "change_frequency";

export interface ApiErrorPayload {
  error: {
    code: string;
    message: string;
  };
}

export interface HealthResponse {
  status: "ok";
  version: string;
}

export interface GitHistoryRange {
  from?: string | null;
  to?: string | null;
}

export interface LengthSummary {
  max: number;
  total: number;
}

export interface Metrics {
  linesOfCode: number;
  functionCount: number;
  classCount: number;
  functionLength: LengthSummary;
  classLength: LengthSummary;
}

export interface Diagnostic {
  code: string;
  message: string;
  severity: "warning" | "error";
}

export interface Declaration {
  name: string;
  kind: "function" | "method" | "class";
  startLine: number;
  endLine: number;
  length: number;
}

export interface FileAnalysis {
  path: string;
  language: SupportedLanguage;
  status: FileAnalysisStatus;
  usedFallback: boolean;
  metrics: Metrics;
  functions: Declaration[];
  classes: Declaration[];
  diagnostics: Diagnostic[];
}

export interface ModuleAnalysis {
  path: string;
  fileCount: number;
  failedFileCount: number;
  metrics: Metrics;
}

export interface BatchAnalysis {
  files: FileAnalysis[];
  modules: ModuleAnalysis[];
}

export interface StaticAnalysisRequest {
  rootPath: string;
  /** Omit to let the backend discover supported sources below rootPath. */
  files?: string[];
}

export interface GitFileMetrics {
  path: string;
  previousPaths: string[];
  deleted: boolean;
  gitChurn: number;
  changeFrequency: number;
  authorCount: number;
  firstChangedAt: string | null;
  lastChangedAt: string | null;
  changeConcentration: number;
  changeDispersion: number;
}

export interface GitAnalysisRequest {
  repositoryPath: string;
  history?: GitHistoryRange;
}

export interface GitAnalysisReport {
  commitCount: number;
  history: GitHistoryRange;
  files: GitFileMetrics[];
  diagnostics: Diagnostic[];
}

export interface HotspotMetrics {
  linesOfCode: number;
  maxFunctionLength: number;
  maxClassLength: number;
  gitChurn: number;
  changeFrequency: number;
}

export interface HotspotInput {
  path: string;
  metrics: HotspotMetrics;
}

export interface RiskBands {
  lowMax: number;
  mediumMax: number;
}

export interface RiskProfile {
  version: string;
  weights: Partial<Record<RiskMetric, number>>;
  bands: RiskBands;
}

export interface RiskContribution {
  metric: RiskMetric;
  raw: number;
  normalizationCap: number;
  normalized: number;
  weight: number;
  weighted: number;
}

export interface Hotspot {
  path: string;
  module: string;
  metrics: HotspotMetrics;
  score: number;
  level: RiskLevel;
  rank: number;
  contributions: RiskContribution[];
}

export interface ModuleHotspot {
  path: string;
  fileCount: number;
  score: number;
  level: RiskLevel;
  highestFileScore: number;
}

export interface HeatmapCell {
  path: string;
  module: string;
  xValue: number;
  yValue: number;
  score: number;
  level: RiskLevel;
  rank: number;
}

export interface HeatmapData {
  xMetric: RiskMetric;
  yMetric: RiskMetric;
  cells: HeatmapCell[];
}

export interface RiskAnalysisRequest {
  files: HotspotInput[];
  profile?: RiskProfile;
  availableMetrics?: RiskMetric[];
}

export interface RiskAnalysisReport {
  profile: RiskProfile;
  requestedProfile: RiskProfile;
  availableMetrics: RiskMetric[];
  unavailableMetrics: RiskMetric[];
  formula: string;
  normalization: string;
  limitations: string[];
  files: Hotspot[];
  modules: ModuleHotspot[];
  heatmap: HeatmapData;
}
