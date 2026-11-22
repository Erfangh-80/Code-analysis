import { useQuery } from "@tanstack/react-query";

import { coreApi } from "@/services/api/core-api";

export function useHealthQuery() {
  return useQuery({
    queryKey: ["local-api", "health"],
    queryFn: ({ signal }) => coreApi.health({ signal }),
    retry: false,
    refetchInterval: 30_000,
    staleTime: 20_000,
  });
}
