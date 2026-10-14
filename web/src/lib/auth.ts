import { create } from "zustand";
import { getToken, setTokens } from "./api/client";
import { useLlmModelCache } from "./llmModelCache";
import { useUI } from "./ui";

interface AuthState {
  authed: boolean;
  session: number;
  signIn: (access: string, refresh: string) => void;
  signOut: () => void;
}

function discardPrivateState() {
  useLlmModelCache.setState({ modelsByBaseUrl: {} });
  useUI.getState().closeModal();
  useUI.setState({
    commandOpen: false,
    mobileNavOpen: false,
    positionSizeOpen: false,
    kellyOpen: false,
    fxOpen: false,
  });
}

function signOut() {
  discardPrivateState();
  setTokens("", "");
  useAuth.setState((s) => ({ authed: false, session: s.session + 1 }));
}

// Reactive auth presence, mirrored from the api client's token storage so the
// shell can gate on it. The api client remains the source of truth for the
// token value; the session counter scopes identity/admin query caches across sign-ins.
export const useAuth = create<AuthState>((set) => ({
  authed: !!getToken(),
  session: 0,
  signIn: (access, refresh) => {
    discardPrivateState();
    setTokens(access, refresh);
    set((s) => ({ authed: true, session: s.session + 1 }));
  },
  signOut,
}));
