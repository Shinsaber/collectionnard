import { describe, expect, it } from "vitest";
import { fitWithin, imageSize, outputName, outputType } from "./images";

describe("fitWithin", () => {
  it("réduit le plus grand côté à max en gardant le ratio", () => {
    expect(fitWithin(4000, 3000, 2000)).toEqual({ width: 2000, height: 1500 });
    expect(fitWithin(3000, 4000, 2000)).toEqual({ width: 1500, height: 2000 });
  });

  it("n'agrandit jamais une image plus petite", () => {
    expect(fitWithin(800, 1200, 2000)).toEqual({ width: 800, height: 1200 });
  });

  it("garde au moins un pixel", () => {
    expect(fitWithin(1, 10000, 100)).toEqual({ width: 1, height: 100 });
    expect(fitWithin(0, 0, 100)).toEqual({ width: 1, height: 1 });
  });
});

describe("outputType / outputName", () => {
  it("garde le PNG et passe tout le reste en JPEG", () => {
    expect(outputType("image/png")).toBe("image/png");
    expect(outputType("image/jpeg")).toBe("image/jpeg");
    expect(outputType("image/heic")).toBe("image/jpeg");
    expect(outputType("")).toBe("image/jpeg");
  });

  it("remplace l'extension selon le type de sortie", () => {
    expect(outputName("IMG_0042.HEIC", "image/jpeg")).toBe("IMG_0042.jpg");
    expect(outputName("cover.png", "image/png")).toBe("cover.png");
    expect(outputName("sans-extension", "image/jpeg")).toBe(
      "sans-extension.jpg",
    );
    expect(outputName(".jpg", "image/jpeg")).toBe("image.jpg");
  });
});

describe("imageSize", () => {
  it("rend null quand le navigateur ne décode pas (jsdom sans createImageBitmap)", async () => {
    expect(await imageSize(new Blob(["x"], { type: "image/png" }))).toBeNull();
  });
});
