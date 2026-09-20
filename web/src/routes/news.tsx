import { createFileRoute } from "@tanstack/react-router";
import { NewsView, type NewsFormValue } from "@/app/screens/NewsView";
import { useToastManager } from "@/components/Toast";
import type { News } from "@/lib/api/news";
import {
  useCreateNews,
  useDeleteNews,
  useNews,
  useUpdateNews,
  usePredictionActions,
} from "@/lib/hooks/useNews";

export const Route = createFileRoute("/news")({ component: NewsPage });

function NewsPage() {
  const predictionActions = usePredictionActions();
  const toast = useToastManager();
  const newsQ = useNews();
  const createM = useCreateNews();
  const updateM = useUpdateNews();
  const deleteM = useDeleteNews();

  const save = async (value: NewsFormValue, item?: News) => {
    if (item) {
      await updateM.mutateAsync({
        id: item.id,
        body: value.body,
        previousAssetIds: item.assets.map((asset) => asset.id),
        assets: value.assets,
      });
      toast.add({ title: "News thesis updated", description: value.body.title });
      return;
    }
    await createM.mutateAsync({ ...value.body, assets: value.assets });
    toast.add({ title: "News thesis created", description: value.body.title });
  };

  return (
    <NewsView
      predictionActions={predictionActions}
      news={newsQ.data ?? []}
      loading={newsQ.isLoading}
      error={newsQ.isError}
      onRetry={() => void newsQ.refetch()}
      onSave={save}
      onDelete={async (item) => {
        await deleteM.mutateAsync(item.id);
        toast.add({ title: "News thesis deleted", description: item.title });
      }}
    />
  );
}
