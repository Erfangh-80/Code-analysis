import { CircleAlert, FileSearch, LoaderCircle, RefreshCw } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/services/api/client";

interface StateFrameProps {
  className?: string;
}

export function LoadingState({ className = "" }: StateFrameProps) {
  return (
    <div className={`flex min-h-44 items-center justify-center gap-2 text-sm text-muted-foreground ${className}`} role="status">
      <LoaderCircle className="size-4 animate-spin" aria-hidden="true" />
      در حال دریافت اطلاعات از سرویس محلی…
    </div>
  );
}

interface EmptyStateProps extends StateFrameProps {
  title: string;
  description: string;
  action?: ReactNode;
}

export function EmptyState({ title, description, action, className = "" }: EmptyStateProps) {
  return (
    <div className={`flex min-h-44 flex-col items-center justify-center rounded-xl border border-dashed bg-muted/35 p-6 text-center ${className}`}>
      <FileSearch className="size-7 text-muted-foreground" aria-hidden="true" />
      <p className="mt-3 text-sm font-extrabold">{title}</p>
      <p className="mt-1 max-w-lg text-xs leading-5 text-muted-foreground">{description}</p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

interface ApiErrorStateProps extends StateFrameProps {
  error: unknown;
  onRetry?: () => void;
  title?: string;
}

export function ApiErrorState({
  error,
  onRetry,
  title = "امکان دریافت اطلاعات وجود ندارد",
  className = "",
}: ApiErrorStateProps) {
  return (
    <div className={`flex min-h-44 flex-col items-center justify-center rounded-xl border border-destructive/25 bg-destructive/7 p-6 text-center ${className}`} role="alert">
      <CircleAlert className="size-7 text-destructive" aria-hidden="true" />
      <p className="mt-3 text-sm font-extrabold">{title}</p>
      <p className="mt-1 max-w-lg text-xs leading-5 text-muted-foreground">{getApiErrorMessage(error)}</p>
      {onRetry && (
        <Button className="mt-4" variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="size-3.5" />
          تلاش دوباره
        </Button>
      )}
    </div>
  );
}
