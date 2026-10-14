import { useAuth } from "@/lib/auth";
import { useMe } from "./useMe";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  type OcrModelsRequest,
  type OcrSettingsPut,
  type OcrSettingsTestRequest,
  settingsApi,
} from "@/lib/api/settings";

export function useOcrSettings() {
  const session = useAuth((s) => s.session);
  const authed = useAuth((s) => s.authed);
  const me = useMe();
  const allowed = authed && me.data?.is_admin === true;
  const query = useQuery({
    queryKey: ["settings", "ocr", session],
    enabled: allowed,
    queryFn: () => settingsApi.getOcrSettings(),
  });
  return {
    ...query,
    data: allowed && !query.isError ? query.data : undefined,
    isPending: authed && (me.isPending || (allowed && query.isPending)),
    isError: query.isError || me.isError || (!me.isPending && !allowed),
  };
}

export function useSaveOcrSettings() {
  const session = useAuth((s) => s.session);
  const qc = useQueryClient();
  return useMutation({
    mutationKey: ["settings", "ocr", "save", session],
    mutationFn: (body: OcrSettingsPut) => settingsApi.putOcrSettings(body),
    onSuccess: (data) => {
      if (useAuth.getState().session === session) {
        qc.setQueryData(["settings", "ocr", session], data);
      }
    },
  });
}

export function useTestOcrSettings() {
  const session = useAuth((s) => s.session);
  return useMutation({
    mutationKey: ["settings", "ocr", "test", session],
    mutationFn: (body: OcrSettingsTestRequest) => settingsApi.testOcrSettings(body),
  });
}

export function useListOcrModels() {
  const session = useAuth((s) => s.session);
  return useMutation({
    mutationKey: ["settings", "ocr", "list", session],
    mutationFn: (body: OcrModelsRequest) => settingsApi.listOcrModels(body),
  });
}
