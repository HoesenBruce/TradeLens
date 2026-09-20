import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  newsApi,
  predictionApi,
  type Prediction,
  type PredictionBody,
  type NewsAssetBody,
  type NewsBody,
} from "@/lib/api/news";

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

export function usePredictionActions() {
  const client = useQueryClient();
  return {
    onSavePrediction: async (newsId: string, body: PredictionBody, prediction?: Prediction) => {
      if (prediction) await predictionApi.update(newsId, prediction.id, body);
      else await predictionApi.create(newsId, body);
      await client.invalidateQueries({ queryKey: ["news"] });
    },
    onDeletePrediction: async (newsId: string, prediction: Prediction) => {
      await predictionApi.delete(newsId, prediction.id);
      await client.invalidateQueries({ queryKey: ["news"] });
    },
  };
}

export function useNewsDetail(id: string) {
  return useQuery({ queryKey: ["news", id], queryFn: () => newsApi.get(id) });
}
