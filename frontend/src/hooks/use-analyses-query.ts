import { useQuery } from "@tanstack/react-query";

import { analysisApi } from "@/services/api/analysis-api";

export function useAnalysesQuery() {
  return useQuery({
    queryKey: ["analyses", { page: 1, pageSize: 10 }],
    queryFn: analysisApi.list,
    staleTime: 10_000,
  });
}
