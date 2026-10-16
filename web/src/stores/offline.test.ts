import { AxiosError } from "axios";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OfflineSnapshot } from "@/api/types";

const { snapshotMock } = vi.hoisted(() => ({ snapshotMock: vi.fn() }));
vi.mock("@/api/client", () => ({ offlineApi: { snapshot: snapshotMock } }));

import { useOfflineStore } from "./offline";

const snap = {
  generated_at: "2026-10-16T12:00:00Z",
  user: { id: "u1", username: "shinsaber" },
  item_types: [],
  collection_types: [],
  collections: [],
  items: [],
  entries: [],
} as unknown as OfflineSnapshot;

describe("useOfflineStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    snapshotMock.mockReset();
  });

  it("garde la copie reçue et lance les étapes suivantes", async () => {
    snapshotMock.mockResolvedValue(snap);
    const store = useOfflineStore();
    const step = vi.fn().mockResolvedValue(undefined);
    store.afterSync.push(step);
    await store.sync();
    expect(store.snapshot).toEqual(snap);
    expect(store.reachable).toBe(true);
    expect(step).toHaveBeenCalledWith(snap);
    // Pas de nouvel appel dans le délai minimal, sauf synchro forcée.
    await store.sync();
    expect(snapshotMock).toHaveBeenCalledTimes(1);
    await store.sync(true);
    expect(snapshotMock).toHaveBeenCalledTimes(2);
  });

  it("passe hors ligne sur une erreur réseau et garde la copie", async () => {
    snapshotMock.mockResolvedValueOnce(snap);
    const store = useOfflineStore();
    await store.sync();
    snapshotMock.mockRejectedValueOnce(
      new AxiosError("Network Error", "ERR_NETWORK"),
    );
    await store.sync(true);
    expect(store.reachable).toBe(false);
    expect(store.snapshot).toEqual(snap);
  });

  it("efface la copie à la déconnexion", async () => {
    snapshotMock.mockResolvedValue(snap);
    const store = useOfflineStore();
    await store.sync();
    await store.clear();
    expect(store.snapshot).toBeNull();
  });
});
