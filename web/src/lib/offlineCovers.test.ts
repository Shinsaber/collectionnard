import { afterEach, describe, expect, it } from "vitest";
import {
  loadCoverSettings,
  planEviction,
  saveCoverSettings,
  thumbUrl,
  wantedThumbs,
} from "./offlineCovers";
import type { OfflineSnapshot } from "@/api/types";

const gallery = {
  properties: {
    images: {
      "x-widget": "image-gallery",
      "x-image-categories": { cover: { ratio: "2/3" } },
    },
  },
};

describe("couvertures hors ligne", () => {
  afterEach(() => localStorage.clear());

  it("retire les moins récemment consultées pour tenir sous le plafond", () => {
    const index = {
      a: { size: 40, used: 3 },
      b: { size: 40, used: 1 },
      c: { size: 40, used: 2 },
    };
    expect(planEviction(index, 200, 50)).toEqual([]);
    expect(planEviction(index, 130, 30)).toEqual(["b"]);
    expect(planEviction(index, 100, 60)).toEqual(["b", "c"]);
    expect(planEviction(index, 50, 60)).toBeNull();
    // Plafond abaissé, sans ajout.
    expect(planEviction(index, 45)).toEqual(["b", "c"]);
  });

  it("ne garde que les vignettes des images envoyées sur le serveur", () => {
    expect(
      thumbUrl(gallery, { images: { cover: ["/api/v1/uploads/a.jpg"] } }),
    ).toBe("/api/v1/uploads/a.jpg?w=240");
    expect(
      thumbUrl(gallery, { images: { cover: ["https://ext/a.jpg"] } }),
    ).toBe("");
    const snap = {
      item_types: [{ id: "t", attributes_schema: gallery }],
      items: [
        {
          item_type_id: "t",
          attributes: { images: { cover: ["/api/v1/uploads/a.jpg"] } },
        },
        {
          item_type_id: "t",
          attributes: { images: { cover: ["/api/v1/uploads/a.jpg"] } },
        },
        { item_type_id: "t", attributes: {} },
      ],
    } as unknown as OfflineSnapshot;
    expect(wantedThumbs(snap)).toEqual(["/api/v1/uploads/a.jpg?w=240"]);
  });

  it("réglages : défauts, valeurs gardées, plafond inconnu ignoré", () => {
    expect(loadCoverSettings()).toEqual({ enabled: true, limitMb: 50 });
    saveCoverSettings({ enabled: false, limitMb: 100 });
    expect(loadCoverSettings()).toEqual({ enabled: false, limitMb: 100 });
    localStorage.setItem("colec-offline-covers", '{"limitMb": 7}');
    expect(loadCoverSettings().limitMb).toBe(50);
  });
});
