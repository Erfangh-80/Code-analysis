import { CircleAlert, LoaderCircle, Wifi } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { useHealthQuery } from "@/hooks/use-health-query";

export function APIConnectionStatus() {
  const health = useHealthQuery();

  if (health.isPending) {
    return (
      <Badge variant="muted">
        <LoaderCircle className="size-3 animate-spin" aria-hidden="true" />
        در حال بررسی سرویس محلی
      </Badge>
    );
  }

  if (health.isError) {
    return (
      <Badge variant="warning">
        <CircleAlert className="size-3" aria-hidden="true" />
        سرویس محلی در دسترس نیست
      </Badge>
    );
  }

  return (
    <Badge variant="success">
      <Wifi className="size-3" aria-hidden="true" />
      سرویس محلی متصل است
    </Badge>
  );
}
