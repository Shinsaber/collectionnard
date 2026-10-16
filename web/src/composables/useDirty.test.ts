import { describe, expect, it } from "vitest";
import { nextTick, reactive } from "vue";
import { formSnapshot, useDirty } from "./useDirty";

describe("formSnapshot", () => {
  it("ignore l'ordre des clés et les valeurs vides", () => {
    expect(formSnapshot({ a: 1, b: "x" })).toBe(formSnapshot({ b: "x", a: 1 }));
    expect(formSnapshot({ a: 1 })).toBe(
      formSnapshot({ a: 1, b: "", c: null, d: [], e: {}, f: { fr: "" } }),
    );
  });
  it("distingue une vraie valeur", () => {
    expect(formSnapshot({ a: 1 })).not.toBe(formSnapshot({ a: 2 }));
    expect(formSnapshot({ t: { fr: "x" } })).not.toBe(formSnapshot({}));
    expect(formSnapshot({ l: [1, 2] })).not.toBe(formSnapshot({ l: [2, 1] }));
  });
});

describe("useDirty", () => {
  it("suit les changements par rapport à l'état de référence", async () => {
    const form = reactive<Record<string, unknown>>({ name: "A" });
    const { dirty, markClean } = useDirty(() => form);
    expect(dirty.value).toBe(false);

    form.name = "B";
    await nextTick();
    expect(dirty.value).toBe(true);

    form.name = "A";
    form.note = "";
    await nextTick();
    expect(dirty.value).toBe(false);

    form.name = "C";
    markClean();
    await nextTick();
    expect(dirty.value).toBe(false);
  });
});
