// Only the Gin gateway is public-facing; the internal Node service authenticates server calls.
export const authorizedFetch = (url: string, options?: RequestInit) => fetch(url, {...options, credentials:'same-origin'});
