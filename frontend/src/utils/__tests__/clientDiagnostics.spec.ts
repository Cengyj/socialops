import { beforeEach, describe, expect, it } from "vitest";
import { recordClientDiagnostic } from "../clientDiagnostics";

describe("recordClientDiagnostic", () => {
  beforeEach(() => {
    delete window.__SOCIALOPS_CLIENT_DIAGNOSTICS__;
  });

  it("records a bounded sanitized diagnostic entry", () => {
    recordClientDiagnostic("payment.fetchConfig", {
      message: "Authorization: Bearer super-secret-token password=plain-text",
      response: {
        status: 403,
        data: {
          code: "payment_disabled",
        },
      },
    });

    const entry = window.__SOCIALOPS_CLIENT_DIAGNOSTICS__?.[0];

    expect(entry).toMatchObject({
      context: "payment.fetchConfig",
      status: 403,
      code: "payment_disabled",
    });
    expect(entry?.at).toEqual(expect.any(String));
    expect(entry?.message).toContain("[redacted]");
    expect(entry?.message).not.toContain("super-secret-token");
    expect(entry?.message).not.toContain("plain-text");
  });

  it("keeps only the latest diagnostics", () => {
    for (let index = 0; index < 90; index += 1) {
      recordClientDiagnostic(`event.${index}`);
    }

    const entries = window.__SOCIALOPS_CLIENT_DIAGNOSTICS__;

    expect(entries).toHaveLength(80);
    expect(entries?.[0]?.context).toBe("event.10");
    expect(entries?.[79]?.context).toBe("event.89");
  });
});
