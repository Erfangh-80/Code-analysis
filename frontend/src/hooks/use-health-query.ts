import { useQuery } from "@tanstack/react-query";

import { analysisApi } from "@/services/api/analysis-api";

export function useHealthQuery() {
  return useQuery({
    queryKey: ["local-api", "health"],
    queryFn: ({ signal }) => analysisApi.health({ signal }),
    retry: false,
    refetchInterval: 30_000,
    staleTime: 20_000,
  });
}
