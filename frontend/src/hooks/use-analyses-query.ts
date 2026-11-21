import { useQuery } from "@tanstack/react-query";

import { analysisApi } from "@/services/api/analysis-api";
import type { AnalysisListParams } from "@/types/api";

export function useAnalysesQuery(params: AnalysisListParams = {}) {
  const normalized = {
    page: params.page ?? 1,
    pageSize: params.pageSize ?? 50,
    status: params.status,
  };

  return useQuery({
    queryKey: ["analyses", normalized],
    queryFn: ({ signal }) => analysisApi.list(normalized, { signal }),
    staleTime: 5_000,
    refetchInterval: (query) => query.state.data?.items.some((analysis) => analysis.status === "queued" || analysis.status === "running") ? 2_000 : false,
    refetchIntervalInBackground: true,
  });
}
