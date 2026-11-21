export type AnalysisRunStatus =
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export type StageStatus =
  | "pending"
  | "running"
  | "completed"
  | "unavailable"
  | "failed"
  | "cancelled"
  | "skipped";

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

export interface AnalysisProgress {
  filesDiscovered: number;
  filesEligible: number;
  filesAnalyzed: number;
  filesWithErrors: number;
  commitsTotal: number;
  commitsScanned: number;
  percent: number;
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
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  progress: AnalysisProgress;
  summary?: AnalysisSummary;
}

export interface AnalysisPage {
  items: AnalysisRun[];
  page: number;
  pageSize: number;
  totalItems: number;
}

export interface AnalysisStartResponse {
  analysis: AnalysisRun;
}
