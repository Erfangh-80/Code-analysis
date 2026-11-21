import { apiRequest } from "@/services/api/client";
import type { AnalysisPage, AnalysisStartResponse, HealthResponse } from "@/types/api";

export const analysisApi = {
  health: () => apiRequest<HealthResponse>("/health"),
  list: () => apiRequest<AnalysisPage>("/analyses?page=1&pageSize=10"),
  start: (projectPath: string) =>
    apiRequest<AnalysisStartResponse>("/analyses", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ projectPath }),
    }),
};
