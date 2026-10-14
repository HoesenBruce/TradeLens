import { useAuth } from "@/lib/auth";
import { useMe } from "./useMe";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { LlmApiModelsRequest, LlmApiSettingsTestRequest } from "@/lib/llmApiSettings";
import { type CoachSettingsPut, settingsApi } from "@/lib/api/settings";

export function useCoachSettings() {
  const session = useAuth((s) => s.session);
  const authed = useAuth((s) => s.authed);
  const me = useMe();
  const allowed = authed && me.data?.is_admin === true;
  const query = useQuery({
    queryKey: ["settings", "coach", session],
    enabled: allowed,
    queryFn: () => settingsApi.getCoachSettings(),
  });
  return {
    ...query,
    data: allowed && !query.isError ? query.data : undefined,
    isPending: authed && (me.isPending || (allowed && query.isPending)),
    isError: query.isError || me.isError || (!me.isPending && !allowed),
  };
}

export function useSaveCoachSettings() {
  const session = useAuth((s) => s.session);
  const qc = useQueryClient();
  return useMutation({
    mutationKey: ["settings", "coach", "save", session],
    mutationFn: (body: CoachSettingsPut) => settingsApi.putCoachSettings(body),
    onSuccess: (data) => {
      if (useAuth.getState().session === session) {
        qc.setQueryData(["settings", "coach", session], data);
      }
    },
  });
}

export function useTestCoachSettings() {
  const session = useAuth((s) => s.session);
  return useMutation({
    mutationKey: ["settings", "coach", "test", session],
    mutationFn: (body: LlmApiSettingsTestRequest) => settingsApi.testCoachSettings(body),
  });
}

export function useListCoachModels() {
  const session = useAuth((s) => s.session);
  return useMutation({
    mutationKey: ["settings", "coach", "list", session],
    mutationFn: (body: LlmApiModelsRequest) => settingsApi.listCoachModels(body),
  });
}
