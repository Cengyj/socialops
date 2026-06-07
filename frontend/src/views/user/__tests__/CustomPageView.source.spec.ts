import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const sourcePath = resolve(__dirname, "../CustomPageView.vue");
const source = readFileSync(sourcePath, "utf8");

describe("CustomPageView localized user-facing states", () => {
  it("keeps markdown page errors and controls localized", () => {
    expect(source).toContain("customPage.toc");
    expect(source).toContain("customPage.copyCode");
    expect(source).toContain("customPage.copied");
    expect(source).toContain("customPage.copyFailed");
    expect(source).toContain("customPage.markdownNotFoundTitle");
    expect(source).toContain("customPage.markdownLoadFailedTitle");
    expect(source).not.toContain("Page not found</p>");
    expect(source).not.toContain("Failed to load page</p>");
  });

  it("keeps nested markdown slugs within one backend path parameter", () => {
    expect(source).toContain("function encodePageSlugParam");
    expect(source).toContain("encodeURIComponent(encodeURIComponent(slug))");
    expect(source).toContain("apiClient.get<string>(`/pages/${encodePageSlugParam(slug)}`");
    expect(source).toContain("responseType: 'text'");
    expect(source).toContain("`/api/v1/pages/${encodePageSlugParam(slug)}/images/");
    expect(source).not.toContain("/pages/${encodeURIComponent(slug)}");
  });
});
