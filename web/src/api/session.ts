// Session perdue (jeton refusé, non renouvelable) : l'intercepteur de client.ts marque l'erreur
// et renvoie vers la connexion ; useNotify ne l'affiche pas (ce serait du bruit).

export function markSessionExpired(error: unknown): void {
  if (error && typeof error === "object")
    (error as { sessionExpired?: boolean }).sessionExpired = true;
}

export function isSessionExpired(e: unknown): boolean {
  return !!(e as { sessionExpired?: boolean } | null)?.sessionExpired;
}
