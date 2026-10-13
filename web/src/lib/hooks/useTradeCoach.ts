import { useCoachAvailability } from "./useCoachAvailability";
import type { StoredTradeCoachReview, TradeCoachApiNote, TradeCoachReview } from "@/lib/api/trades";

export const tradeCoachReviewsKey = (tradeId: string) => ["trade-coach-reviews", tradeId] as const;

// Legacy cache entries are never renderable in this build, even if settings say enabled.
export function useTradeCoach(_tradeId: string | undefined) {
  const capability = useCoachAvailability();
  return {
    coachConfigured: false,
    settingsPending: capability.isPending,
    unavailable: true,
    isPending: false,
    isError: false,
    data: undefined as TradeCoachReview | undefined,
    streamingNotes: [] as TradeCoachApiNote[],
    fromStorage: false,
    history: [] as StoredTradeCoachReview[],
    error: null,
    generate: async (): Promise<TradeCoachReview> => {
      throw new Error("AI Coach is unavailable.");
    },
    reset: () => {},
  };
}
