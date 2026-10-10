import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { backupApi, backupNeedsAttention } from "@/lib/api/backup";
import { useAuth } from "@/lib/auth";
import { useMe } from "./useMe";

const backupKey = (session: number) => ["admin", "backup", session];

/**
 * Snapshot status. Owner-only — the API answers 403 to anyone else, so callers
 * gate the query on `me.is_admin`.
 */
export function useBackupStatus(enabled: boolean) {
  const session = useAuth((s) => s.session);
  return useQuery({
    queryKey: backupKey(session),
    queryFn: () => backupApi.status(),
    enabled,
    // The schedule runs server-side; a slow poll keeps the shell dot honest
    // without the owner reloading.
    refetchInterval: 5 * 60_000,
  });
}

export function useRunBackup() {
  const session = useAuth((s) => s.session);
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => backupApi.run(),
    onSuccess: (status) => qc.setQueryData(backupKey(session), status),
    // A failed run records its error server-side; refetch so it shows.
    onError: () => qc.invalidateQueries({ queryKey: backupKey(session) }),
  });
}

/**
 * Backup health for the app shell: true when the last snapshot attempt failed
 * or the newest snapshot is overdue. Like a dead broker sync, a backup that
 * quietly stopped looks identical to one that works until the day it's needed.
 * Members never ask (the endpoint is owner-only).
 */
export function useBackupAttention(): boolean {
  const me = useMe();
  const { data } = useBackupStatus(me.data?.is_admin ?? false);
  return Boolean(me.data?.is_admin) && backupNeedsAttention(data);
}
