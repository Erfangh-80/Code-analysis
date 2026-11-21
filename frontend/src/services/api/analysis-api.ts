import { apiDownload, apiRequest, buildQuery, type ApiRequestOptions } from "@/services/api/client";
import type {
  AnalysisEnvelope,
  AnalysisFilePage,
  AnalysisFileResult,
  AnalysisListParams,
  AnalysisModules,
  AnalysisPage,
  AnalysisReport,
  AnalysisRequest,
  AnalysisResultSummary,
  AnalysisStartResponse,
  AnalysisHotspotPage,
  HeatmapData,
  HealthResponse,
  ResultListParams,
} from "@/types/api";

function resultQuery(params: ResultListParams = {}) {
  return buildQuery({
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 50,
    q: params.q,
    module: params.module,
    riskLevel: params.riskLevel,
    analysisStatus: params.analysisStatus,
    sort: params.sort,
    direction: params.direction,
  });
}

export const analysisApi = {
  health: (options?: ApiRequestOptions) => apiRequest<HealthResponse>("/health", options),

  list: (params: AnalysisListParams = {}, options?: ApiRequestOptions) =>
    apiRequest<AnalysisPage>(
      `/analyses${buildQuery({ page: params.page ?? 1, pageSize: params.pageSize ?? 50, status: params.status })}`,
      options,
    ),

  start: (request: AnalysisRequest, options?: ApiRequestOptions) =>
    apiRequest<AnalysisStartResponse>("/analyses", {
      ...options,
      method: "POST",
      timeoutMs: 30_000,
      headers: { "Content-Type": "application/json", ...options?.headers },
      body: JSON.stringify(request),
    }),

  get: (analysisId: string, options?: ApiRequestOptions) =>
    apiRequest<AnalysisEnvelope>(`/analyses/${encodeURIComponent(analysisId)}`, options),

  cancel: (analysisId: string, options?: ApiRequestOptions) =>
    apiRequest<AnalysisEnvelope>(`/analyses/${encodeURIComponent(analysisId)}/cancel`, {
      ...options,
      method: "POST",
    }),

  summary: (analysisId: string, options?: ApiRequestOptions) =>
    apiRequest<AnalysisResultSummary>(`/analyses/${encodeURIComponent(analysisId)}/summary`, options),

  hotspots: (analysisId: string, params: ResultListParams = {}, options?: ApiRequestOptions) =>
    apiRequest<AnalysisHotspotPage>(`/analyses/${encodeURIComponent(analysisId)}/hotspots${resultQuery(params)}`, options),

  files: (analysisId: string, params: ResultListParams = {}, options?: ApiRequestOptions) =>
    apiRequest<AnalysisFilePage>(`/analyses/${encodeURIComponent(analysisId)}/files${resultQuery(params)}`, options),

  file: (analysisId: string, path: string, options?: ApiRequestOptions) =>
    apiRequest<AnalysisFileResult>(
      `/analyses/${encodeURIComponent(analysisId)}/files/detail${buildQuery({ path })}`,
      options,
    ),

  modules: (analysisId: string, options?: ApiRequestOptions) =>
    apiRequest<AnalysisModules>(`/analyses/${encodeURIComponent(analysisId)}/modules`, options),

  heatmap: (analysisId: string, options?: ApiRequestOptions) =>
    apiRequest<HeatmapData>(`/analyses/${encodeURIComponent(analysisId)}/heatmap`, options),

  report: (analysisId: string, options?: ApiRequestOptions) =>
    apiRequest<AnalysisReport>(`/analyses/${encodeURIComponent(analysisId)}/report`, options),

  export: (analysisId: string, format: "json" | "csv", options?: ApiRequestOptions) =>
    apiDownload(`/analyses/${encodeURIComponent(analysisId)}/export${buildQuery({ format })}`, options),
};
