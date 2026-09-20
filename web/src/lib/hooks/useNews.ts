import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { newsApi, type NewsAssetBody, type NewsBody } from "@/lib/api/news";

export type NewsAssetDraft = NewsAssetBody & { id?: string };

export function useNews() {
  return useQuery({ queryKey: ["news"], queryFn: newsApi.list });
}

export function useCreateNews() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: newsApi.create,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["news"] }),
  });
}

export function useUpdateNews() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      body,
      previousAssetIds,
      assets,
    }: {
      id: string;
      body: NewsBody;
      previousAssetIds: string[];
      assets: NewsAssetDraft[];
    }) => {
      await newsApi.update(id, body);
      const kept = new Set(assets.flatMap((asset) => (asset.id ? [asset.id] : [])));
      await Promise.all([
        ...assets.map(({ id: assetId, ...asset }) =>
          assetId ? newsApi.updateAsset(id, assetId, asset) : newsApi.createAsset(id, asset),
        ),
        ...previousAssetIds
          .filter((assetId) => !kept.has(assetId))
          .map((assetId) => newsApi.deleteAsset(id, assetId)),
      ]);
      return newsApi.get(id);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["news"] }),
  });
}

export function useDeleteNews() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: newsApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["news"] }),
  });
}
