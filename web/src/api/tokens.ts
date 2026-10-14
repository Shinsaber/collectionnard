// Stockage des tokens (localStorage). Seul point d'accès : le store auth et le client HTTP.
// Tout accès est protégé : localStorage peut être indisponible (navigation privée, quotas).

const ACCESS_KEY = "token";
const REFRESH_KEY = "refresh_token";

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function getAccessToken(): string | null {
  return read(ACCESS_KEY);
}

export function getRefreshToken(): string | null {
  return read(REFRESH_KEY);
}

export function setTokens(access: string, refresh: string): void {
  try {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  } catch {
    /* stockage indisponible : la session ne survivra pas au rechargement */
  }
}

export function clearTokens(): void {
  try {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  } catch {
    /* ignoré */
  }
}
