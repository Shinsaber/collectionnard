import { beforeEach, describe, expect, it, vi } from "vitest";

const message = vi.hoisted(() => vi.fn());
vi.mock("element-plus", () => ({ ElMessage: message }));

import { useNotify } from "./useNotify";

describe("useNotify", () => {
  beforeEach(() => message.mockReset());

  it("n'affiche rien pour une session perdue (redirection vers la connexion)", () => {
    const { notifyError } = useNotify();
    notifyError(Object.assign(new Error("401"), { sessionExpired: true }));
    expect(message).not.toHaveBeenCalled();
    notifyError(new Error("boom"));
    expect(message).toHaveBeenCalledTimes(1);
  });
});
