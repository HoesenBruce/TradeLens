import { create } from "zustand";
import { getToken, setTokens } from "./api/client";

interface AuthState {
  authed: boolean;
  session: number;
  signIn: (access: string, refresh: string) => void;
  signOut: () => void;
}

function signOut() {
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
    setTokens(access, refresh);
    set((s) => ({ authed: true, session: s.session + 1 }));
  },
  signOut,
}));
