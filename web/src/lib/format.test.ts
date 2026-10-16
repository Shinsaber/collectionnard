import { describe, expect, it } from "vitest";
import { formatBytes } from "./format";

describe("formatBytes", () => {
  it("choisit l'unité et suit la langue", () => {
    expect(formatBytes(512, "en")).toBe("512 byte");
    expect(formatBytes(1536, "en")).toBe("1.5 kB");
    expect(formatBytes(3 * 1024 * 1024, "en")).toBe("3 MB");
    expect(formatBytes(1536, "fr")).toMatch(/^1,5\s?ko$/);
  });
});
