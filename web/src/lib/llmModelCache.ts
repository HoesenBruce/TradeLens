import { create } from "zustand";

export const LLM_MODEL_CACHE_STORAGE_KEY = "tm-llm-model-cache";

/** Keep the most recently fetched providers only. */
const MAX_CACHED_BASE_URLS = 8;

interface LlmModelCacheState {
  /** Fetched model ids per API base URL, so lists survive navigation. */
  modelsByBaseUrl: Record<string, string[]>;
  setModels: (baseUrl: string, models: string[]) => void;
}

// Provider URLs and discovered model names are private configuration, not preferences.
try {
  localStorage.removeItem(LLM_MODEL_CACHE_STORAGE_KEY);
} catch {
  /* storage may be unavailable */
}

export const useLlmModelCache = create<LlmModelCacheState>()((set) => ({
  modelsByBaseUrl: {},
  setModels: (baseUrl, models) =>
    set((s) => {
      const { [baseUrl]: _, ...rest } = s.modelsByBaseUrl;
      const entries = Object.entries(rest).slice(-(MAX_CACHED_BASE_URLS - 1));
      return { modelsByBaseUrl: { ...Object.fromEntries(entries), [baseUrl]: models } };
    }),
}));
