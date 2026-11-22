import { CircleAlert, LoaderCircle, Wifi } from "lucide-react";

import { useHealthQuery } from "@/hooks/use-health-query";

const baseClassName = "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-extrabold";

export function APIConnectionStatus() {
  const health = useHealthQuery();

  if (health.isPending) {
    return <span className={`${baseClassName} bg-muted text-muted-foreground`}><LoaderCircle className="size-3 animate-spin" aria-hidden="true" />در حال بررسی سرویس محلی</span>;
  }
  if (health.isError) {
    return <span className={`${baseClassName} bg-warning/10 text-[color:var(--warning)]`}><CircleAlert className="size-3" aria-hidden="true" />سرویس محلی در دسترس نیست</span>;
  }
  return <span className={`${baseClassName} bg-success/10 text-[color:var(--success)]`}><Wifi className="size-3" aria-hidden="true" />سرویس محلی متصل است</span>;
}
