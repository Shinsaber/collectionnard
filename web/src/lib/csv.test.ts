import { describe, expect, it } from "vitest";
import { defaultDelimiter } from "./csv";

describe("defaultDelimiter", () => {
  it("point-virgule pour le français, virgule sinon", () => {
    expect(defaultDelimiter("fr")).toBe("semicolon");
    expect(defaultDelimiter("fr-CA")).toBe("semicolon");
    expect(defaultDelimiter("en")).toBe("comma");
    expect(defaultDelimiter("de")).toBe("comma");
  });
});
