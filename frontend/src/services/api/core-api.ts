import { apiRequest, type ApiRequestOptions } from "@/services/api/client";
import type {
  BatchAnalysis,
  GitAnalysisReport,
  GitAnalysisRequest,
  HealthResponse,
  RiskAnalysisReport,
  RiskAnalysisRequest,
  StaticAnalysisRequest,
} from "@/types/api";

export const coreApi = {
  health: (options?: ApiRequestOptions) => apiRequest<HealthResponse>("/health", options),

  staticAnalysis: (request: StaticAnalysisRequest, options?: ApiRequestOptions) =>
    apiRequest<BatchAnalysis>("/static-analysis", {
      ...options,
      method: "POST",
      timeoutMs: 30_000,
      headers: { "Content-Type": "application/json", ...options?.headers },
      body: JSON.stringify(request),
    }),

  gitAnalysis: (request: GitAnalysisRequest, options?: ApiRequestOptions) =>
    apiRequest<GitAnalysisReport>("/git-analysis", {
      ...options,
      method: "POST",
      timeoutMs: 60_000,
      headers: { "Content-Type": "application/json", ...options?.headers },
      body: JSON.stringify(request),
    }),

  riskAnalysis: (request: RiskAnalysisRequest, options?: ApiRequestOptions) =>
    apiRequest<RiskAnalysisReport>("/risk-analysis", {
      ...options,
      method: "POST",
      timeoutMs: 30_000,
      headers: { "Content-Type": "application/json", ...options?.headers },
      body: JSON.stringify(request),
    }),
};
