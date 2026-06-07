import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const sourceRoot = resolve(__dirname, "../..");

function readSource(relativePath: string): string {
  return readFileSync(resolve(sourceRoot, relativePath), "utf8");
}

describe("client diagnostics production console contract", () => {
  const criticalSources = [
    "App.vue",
    "components/common/DataTable.vue",
    "components/common/SubscriptionProgressMini.vue",
    "components/layout/AppHeader.vue",
    "composables/usePersistedPageSize.ts",
    "router/index.ts",
    "stores/adminSettings.ts",
    "stores/announcements.ts",
    "stores/app.ts",
    "stores/auth.ts",
    "stores/payment.ts",
    "stores/subscriptions.ts",
  ];

  it("keeps expected network and persistence failures out of the browser error console", () => {
    for (const relativePath of criticalSources) {
      const source = readSource(relativePath);

      expect(source, relativePath).not.toContain("console.error");
      expect(source, relativePath).not.toContain("console.warn");
      expect(source, relativePath).not.toContain("console.log");
      expect(source, relativePath).toContain("recordClientDiagnostic");
    }
  });

  it("uses a sanitized diagnostics buffer instead of raw console output", () => {
    const source = readSource("utils/clientDiagnostics.ts");

    expect(source).toContain("recordClientDiagnostic");
    expect(source).toContain("__SOCIALOPS_CLIENT_DIAGNOSTICS__");
    expect(source).toContain("redactSensitiveText");
    expect(source).not.toContain("console.error");
    expect(source).not.toContain("console.warn");
    expect(source).not.toContain("console.log");
  });
});
