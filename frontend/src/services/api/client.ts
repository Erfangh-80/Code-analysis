import type { ApiErrorPayload } from "@/types/api";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";
const defaultTimeoutMs = 20_000;

export class ApiRequestError extends Error {
  readonly status: number;
  readonly code: string;
  readonly retryable: boolean;

  constructor(status: number, code: string, message: string, retryable = false) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = code;
    this.retryable = retryable;
  }
}

export interface ApiRequestOptions extends RequestInit {
  timeoutMs?: number;
}

function createRequestSignal(signal: AbortSignal | null | undefined, timeoutMs: number) {
  const controller = new AbortController();
  let timedOut = false;
  const abortFromCaller = () => controller.abort(signal?.reason);

  if (signal?.aborted) {
    abortFromCaller();
  } else {
    signal?.addEventListener("abort", abortFromCaller, { once: true });
  }

  const timeout = window.setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  return {
    signal: controller.signal,
    didTimeout: () => timedOut,
    dispose: () => {
      window.clearTimeout(timeout);
      signal?.removeEventListener("abort", abortFromCaller);
    },
  };
}

function isRetryableStatus(status: number) {
  return status === 0 || status === 408 || status === 429 || status >= 500;
}

async function errorFromResponse(response: Response) {
  const payload = (await response.json().catch(() => null)) as ApiErrorPayload | null;
  return new ApiRequestError(
    response.status,
    payload?.error.code ?? "REQUEST_FAILED",
    payload?.error.message ?? "پاسخ قابل‌خواندن از سرویس محلی دریافت نشد.",
    isRetryableStatus(response.status),
  );
}

async function request(path: string, init: ApiRequestOptions = {}) {
  const { timeoutMs = defaultTimeoutMs, signal: callerSignal, headers, ...requestInit } = init;
  const timeout = createRequestSignal(callerSignal, timeoutMs);

  try {
    const requestHeaders = new Headers(headers);
    if (!requestHeaders.has("Accept")) {
      requestHeaders.set("Accept", "application/json");
    }
    const response = await fetch(`${apiBaseUrl}${path}`, {
      ...requestInit,
      signal: timeout.signal,
      headers: requestHeaders,
    });

    if (!response.ok) {
      throw await errorFromResponse(response);
    }

    return response;
  } catch (error) {
    if (error instanceof ApiRequestError) {
      throw error;
    }
    if (timeout.didTimeout()) {
      throw new ApiRequestError(408, "REQUEST_TIMEOUT", "پاسخ سرویس محلی بیش از حد طول کشید. دوباره تلاش کنید.", true);
    }
    if (callerSignal?.aborted) {
      throw new ApiRequestError(499, "REQUEST_CANCELLED", "درخواست لغو شد.");
    }
    throw new ApiRequestError(0, "LOCAL_API_UNREACHABLE", "ارتباط با سرویس محلی برقرار نشد. اجرا بودن سرویس را بررسی کنید.", true);
  } finally {
    timeout.dispose();
  }
}

export async function apiRequest<T>(path: string, init?: ApiRequestOptions): Promise<T> {
  const response = await request(path, init);
  return response.json() as Promise<T>;
}

export async function apiDownload(path: string, init?: ApiRequestOptions) {
  const response = await request(path, init);
  return {
    blob: await response.blob(),
    contentDisposition: response.headers.get("Content-Disposition"),
    contentType: response.headers.get("Content-Type"),
  };
}

export function buildQuery(params: Record<string, string | number | string[] | undefined>) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === "") {
      continue;
    }
    if (Array.isArray(value)) {
      value.forEach((item) => query.append(key, item));
    } else {
      query.set(key, String(value));
    }
  }
  const serialized = query.toString();
  return serialized ? `?${serialized}` : "";
}

export function getApiErrorMessage(error: unknown, fallback = "امکان دریافت داده از سرویس محلی وجود ندارد.") {
  return error instanceof ApiRequestError ? error.message : fallback;
}
