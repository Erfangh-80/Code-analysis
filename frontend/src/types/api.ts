export type AnalysisRunStatus = "queued" | "running" | "completed" | "failed" | "cancelled";

export type AnalysisStage = "validation" | "discovery" | "static_analysis" | "git_analysis" | "risk_scoring";

export type StageStatus =
  | "pending"
  | "running"
  | "completed"
  | "unavailable"
  | "failed"
  | "cancelled"
  | "skipped";

export type FileAnalysisStatus =
  | "analyzed"
  | "empty"
  | "no_executable_code"
  | "fallback"
  | "parse_error"
  | "unsupported"
  | "read_error";

export type SupportedLanguage = "go" | "javascript" | "python" | "unknown";
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

export interface RiskBands {
  lowMax: number;
  mediumMax: number;
}

export interface RiskProfile {
  version: string;
  weights: Partial<Record<RiskMetric, number>>;
  bands: RiskBands;
}

export interface AnalysisRequest {
  projectPath: string;
  history?: GitHistoryRange;
  profile?: RiskProfile;
}

export interface StageProgress {
  total: number;
  completed: number;
}

export interface StageSnapshot {
  stage: AnalysisStage;
  status: StageStatus;
  startedAt?: string | null;
  completedAt?: string | null;
  progress: StageProgress;
}

export interface AnalysisProgress {
  filesDiscovered: number;
  filesEligible: number;
  filesAnalyzed: number;
  filesWithErrors: number;
  commitsTotal: number;
  commitsScanned: number;
  percent: number;
}

export interface AnalysisDiagnostic {
  stage: AnalysisStage;
  path?: string;
  code: string;
  severity: "warning" | "error";
  message: string;
  recordedAt: string;
}

export interface AnalysisSummary {
  filesDiscovered: number;
  filesAnalyzed: number;
  filesWithErrors: number;
  gitStatus: StageStatus;
  riskStatus: StageStatus;
  scoredFiles: number;
  unscoredFiles: number;
}

export interface AnalysisRun {
  id: string;
  status: AnalysisRunStatus;
  currentStage?: AnalysisStage | null;
  createdAt: string;
  startedAt?: string | null;
  completedAt?: string | null;
  history: GitHistoryRange;
  stages: StageSnapshot[];
  progress: AnalysisProgress;
  diagnostics: AnalysisDiagnostic[];
  summary?: AnalysisSummary;
}

export interface AnalysisEnvelope {
  analysis: AnalysisRun;
}

export interface AnalysisPage {
  items: AnalysisRun[];
  page: number;
  pageSize: number;
  totalItems: number;
}

export type AnalysisStartResponse = AnalysisEnvelope;

export interface RiskLevelCounts {
  low: number;
  medium: number;
  high: number;
}

export interface RiskCoverage {
  availableMetrics: RiskMetric[];
  unavailableMetrics: RiskMetric[];
  requestedProfile: RiskProfile;
  effectiveProfile: RiskProfile;
}

export interface AnalysisResultSummary {
  analysis: AnalysisRun;
  summary: AnalysisSummary;
  riskLevels: RiskLevelCounts;
  moduleCount: number;
  riskCoverage?: RiskCoverage;
  hasGitArtifact: boolean;
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

export interface Declaration {
  name: string;
  kind: "function" | "method" | "class";
  startLine: number;
  endLine: number;
  length: number;
}

export interface Diagnostic {
  code: string;
  message: string;
  severity: "warning" | "error";
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

export interface HotspotMetrics {
  linesOfCode?: number;
  maxFunctionLength?: number;
  maxClassLength?: number;
  gitChurn?: number;
  changeFrequency?: number;
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

export interface AnalysisFileResult {
  analysis: FileAnalysis;
  git?: GitFileMetrics;
  risk?: Hotspot;
}

export interface PageResult {
  page: number;
  pageSize: number;
  totalItems: number;
  riskCoverage?: RiskCoverage;
}

export interface AnalysisFilePage extends PageResult {
  items: AnalysisFileResult[];
}

export interface AnalysisHotspotPage extends PageResult {
  items: Hotspot[];
}

export interface AnalysisModules {
  static: ModuleAnalysis[];
  risk: ModuleHotspot[];
  riskCoverage?: RiskCoverage;
}

export interface GitAnalysisReport {
  commitCount: number;
  history: GitHistoryRange;
  files: GitFileMetrics[];
  diagnostics: Diagnostic[];
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

export interface BatchAnalysis {
  files: FileAnalysis[];
  modules: ModuleAnalysis[];
}

export interface AnalysisArtifacts {
  static?: BatchAnalysis;
  git?: GitAnalysisReport;
  risk?: RiskAnalysisReport;
  riskCoverage?: RiskCoverage;
}

export interface AnalysisReport {
  analysis: AnalysisRun;
  artifacts: AnalysisArtifacts;
}

export interface AnalysisListParams {
  page?: number;
  pageSize?: number;
  status?: AnalysisRunStatus;
}

export interface ResultListParams {
  page?: number;
  pageSize?: number;
  q?: string;
  module?: string;
  riskLevel?: RiskLevel[];
  analysisStatus?: FileAnalysisStatus[];
  sort?: string;
  direction?: "asc" | "desc";
}
