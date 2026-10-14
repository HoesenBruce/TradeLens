import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import { useAuth } from "@/lib/auth";

/** A fresh cache and mounted tree for every authentication boundary. */
export function SessionQueryProvider({ children }: { children: ReactNode }) {
  const session = useAuth((s) => s.session);
  return <SessionQueries key={session}>{children}</SessionQueries>;
}

function SessionQueries({ children }: { children: ReactNode }) {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } }),
  );
  useEffect(() => {
    const unsubscribe = useAuth.subscribe((next, previous) => {
      if (next.session !== previous.session) client.clear();
    });
    return () => {
      unsubscribe();
      client.clear();
    };
  }, [client]);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
