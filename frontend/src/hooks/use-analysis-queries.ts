import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { analysisApi } from "@/services/api/analysis-api";
import type { AnalysisRequest, ResultListParams } from "@/types/api";

const activeStatuses = new Set(["queued", "running"]);

export function useAnalysisQuery(analysisId: string | undefined) {
  return useQuery({
    queryKey: ["analysis", analysisId],
    enabled: Boolean(analysisId),
    queryFn: ({ signal }) => analysisApi.get(analysisId!, { signal }),
    staleTime: 0,
    refetchInterval: (query) => (activeStatuses.has(query.state.data?.analysis.status ?? "") ? 1_250 : false),
    refetchIntervalInBackground: true,
  });
}

export function useAnalysisSummaryQuery(analysisId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ["analysis", analysisId, "summary"],
    enabled: Boolean(analysisId) && enabled,
    queryFn: ({ signal }) => analysisApi.summary(analysisId!, { signal }),
    staleTime: 15_000,
  });
}

export function useAnalysisFilesQuery(analysisId: string | undefined, params: ResultListParams = {}, enabled = true) {
  return useQuery({
    queryKey: ["analysis", analysisId, "files", params],
    enabled: Boolean(analysisId) && enabled,
    queryFn: ({ signal }) => analysisApi.files(analysisId!, params, { signal }),
    staleTime: 10_000,
  });
}

export function useAnalysisFileQuery(analysisId: string | undefined, path: string | undefined) {
  return useQuery({
    queryKey: ["analysis", analysisId, "file", path],
    enabled: Boolean(analysisId && path),
    queryFn: ({ signal }) => analysisApi.file(analysisId!, path!, { signal }),
    staleTime: 30_000,
  });
}

export function useAnalysisHotspotsQuery(analysisId: string | undefined, params: ResultListParams = {}, enabled = true) {
  return useQuery({
    queryKey: ["analysis", analysisId, "hotspots", params],
    enabled: Boolean(analysisId) && enabled,
    queryFn: ({ signal }) => analysisApi.hotspots(analysisId!, params, { signal }),
    staleTime: 10_000,
  });
}

export function useAnalysisHeatmapQuery(analysisId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ["analysis", analysisId, "heatmap"],
    enabled: Boolean(analysisId) && enabled,
    queryFn: ({ signal }) => analysisApi.heatmap(analysisId!, { signal }),
    staleTime: 30_000,
  });
}

export function useAnalysisModulesQuery(analysisId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ["analysis", analysisId, "modules"],
    enabled: Boolean(analysisId) && enabled,
    queryFn: ({ signal }) => analysisApi.modules(analysisId!, { signal }),
    staleTime: 30_000,
  });
}

export function useAnalysisReportQuery(analysisId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ["analysis", analysisId, "report"],
    enabled: Boolean(analysisId) && enabled,
    queryFn: ({ signal }) => analysisApi.report(analysisId!, { signal }),
    staleTime: 30_000,
  });
}

export function useValidateProjectPathMutation() {
  return useMutation({
    mutationFn: (projectPath: string) => analysisApi.validateProjectPath(projectPath),
  });
}

export function useStartAnalysisMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (request: AnalysisRequest) => analysisApi.start(request),
    onSuccess: (data) => {
      queryClient.setQueryData(["analysis", data.analysis.id], data);
      return queryClient.invalidateQueries({ queryKey: ["analyses"] });
    },
  });
}

export function useCancelAnalysisMutation(analysisId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => analysisApi.cancel(analysisId),
    onSuccess: (data) => {
      queryClient.setQueryData(["analysis", analysisId], data);
      return queryClient.invalidateQueries({ queryKey: ["analyses"] });
    },
  });
}
