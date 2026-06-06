import { describe, expect, it } from "vitest";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, resolve } from "node:path";

const root = resolve(__dirname, "../..");
const checkedRoots = [
  resolve(root, "views/auth"),
  resolve(root, "views/user"),
  resolve(root, "components/auth"),
  resolve(root, "components/user"),
  resolve(root, "components/TurnstileWidget.vue"),
];

function collectSourceFiles(path: string): string[] {
  if (statSync(path).isFile()) return [path];
  return readdirSync(path)
    .flatMap((entry) => {
      const fullPath = join(path, entry);
      if (entry === "__tests__") return [];
      if (statSync(fullPath).isDirectory()) return collectSourceFiles(fullPath);
      return /\.(vue|ts)$/.test(entry) ? [fullPath] : [];
    });
}

describe("user-facing client diagnostics", () => {
  it("does not write caught user errors directly to the browser console", () => {
    const offenders = checkedRoots
      .flatMap(collectSourceFiles)
      .filter((file) => readFileSync(file, "utf8").includes("console.error"));

    expect(offenders.map((file) => relative(root, file))).toEqual([]);
  });

  it("does not expose raw auth backend errors in user-facing auth surfaces", () => {
    const forbiddenPatterns = [
      /response\?\.data\?\.detail/,
      /response\?\.data\?\.message\s*\|\|\s*err\.message/,
      /err\.message\s*\|\|\s*t\(/,
      /buildAuthErrorMessage\(/,
    ];
    const offenders = checkedRoots
      .flatMap(collectSourceFiles)
      .filter((file) => {
        if (!file.includes(`${resolve(root, "views/auth")}`) && !file.includes(`${resolve(root, "components/auth")}`)) {
          return false;
        }
        const source = readFileSync(file, "utf8");
        return forbiddenPatterns.some((pattern) => pattern.test(source));
      });

    expect(offenders.map((file) => relative(root, file))).toEqual([]);
  });

  it("does not expose raw backend errors in user-facing profile surfaces", () => {
    const forbiddenPatterns = [
      /response\?\.data\?\.detail/,
      /response\?\.data\?\.message/,
      /\berr\.message\b/,
      /\berror\.message\b/,
      /extractApiErrorMessage\(/,
      /extractI18nErrorMessage\(/,
      /buildAuthErrorMessage\(/,
    ];

    const profileRoot = resolve(root, "components/user/profile");
    const offenders = collectSourceFiles(profileRoot).filter((file) => {
      const source = readFileSync(file, "utf8");
      return forbiddenPatterns.some((pattern) => pattern.test(source));
    });

    expect(offenders.map((file) => relative(root, file))).toEqual([]);
  });
});
