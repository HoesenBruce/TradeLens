// API bases can have a proxy prefix or a different origin than the Web app.
export function isApiPath(pathname: string): boolean {
  return (
    /\/api(?:\/|$)/i.test(pathname) ||
    /\/(?:capabilities\/coach|settings\/(?:coach|ocr|alert-channels)(?:\/|$)|trades\/[^/]+\/coach(?:\/|$)|accounts\/[^/]+\/flex-sync(?:\/|$)|flex-sync(?:\/|$)|news\/[^/]+\/analyze$)/.test(
      pathname,
    ) ||
    /\/news\/(?:[^/]+\/)?export$/.test(pathname)
  );
}

export function isNetworkOnlyRequest(request: Request): boolean {
  return request.headers.has("Authorization") || isApiPath(new URL(request.url).pathname);
}

export function isRetiredApiEntry(request: Request): boolean {
  return isNetworkOnlyRequest(request) || new URL(request.url).searchParams.has("__tm_auth_scope");
}

// Older workers put custom API bases in these general-purpose caches too.
export const retiredApiCaches = new Set(["api-cache", "apis", "others"]);
