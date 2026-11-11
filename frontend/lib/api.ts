import type { Analysis, Evaluation, FileResult, HeatCell, ModuleResult, Weights } from "./types";

let token = "";
export function setApiToken(next: string) { token = next; }
async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, { ...options, headers: { "Content-Type": "application/json", ...(token ? { "X-Analyzer-Token": token } : {}), ...(options?.headers ?? {}) } });
  if (!response.ok) { const body = await response.json().catch(() => null); throw new Error(body?.error?.message ?? "ارتباط با API ناموفق بود."); }
  return response.json() as Promise<T>;
}
export const api = {
  analyses: () => request<{ items: Analysis[] }>("/analyses"),
  analysis: (id: string) => request<Analysis>(`/analyses/${id}`),
  createAnalysis: (body: { repository_path: string; ref?: string; languages: string[]; history: { since?: string; until?: string }; weights: Weights }) => request<{ analysis_id: string; status: string }>("/analyses", { method: "POST", body: JSON.stringify(body) }),
  files: (id: string, query = "") => request<{ items: FileResult[]; total: number }>(`/analyses/${id}/files${query}`),
  file: (analysis: string, id: string) => request<FileResult>(`/analyses/${analysis}/files/${id}`),
  modules: (id: string) => request<{ items: ModuleResult[] }>(`/analyses/${id}/modules`),
  heatmap: (id: string, level: "file" | "module" = "file") => request<{ cells: HeatCell[]; bins: number }>(`/analyses/${id}/heatmap?level=${level}&bins=10`),
  createEvaluation: (id: string, history: { since?: string; until?: string }) => request<{ evaluation_id: string; status: string }>(`/analyses/${id}/evaluations`, { method: "POST", body: JSON.stringify({ history }) }),
  evaluation: (analysis: string, evaluation: string) => request<Evaluation>(`/analyses/${analysis}/evaluations/${evaluation}`),
  downloadReport: async (analysis: string, format: "json" | "csv", level: "file" | "module") => {
    const response = await fetch(`/api/v1/analyses/${analysis}/reports?format=${format}&level=${level}`, { headers: token ? { "X-Analyzer-Token": token } : {} });
    if (!response.ok) throw new Error("دریافت گزارش ناموفق بود.");
    return response.blob();
  },
};
