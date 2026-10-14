import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { useAuth } from "@/lib/auth";

export function useCoachAvailability() {
  const session = useAuth((s) => s.session);
  return useQuery({
    queryKey: ["coach-availability-v2", session],
    queryFn: () =>
      apiFetch<{ available: boolean; news_analysis_available: boolean; reason: string }>(
        "/capabilities/coach",
      ),
    staleTime: 0,
  });
}
