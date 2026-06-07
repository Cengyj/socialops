import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const mockApiSource = readFileSync(
  resolve(__dirname, "../../../../../tools/mock-api.mjs"),
  "utf8",
);
const settingsApiSource = readFileSync(
  resolve(__dirname, "../../../api/admin/settings.ts"),
  "utf8",
);
const appStoreSource = readFileSync(
  resolve(__dirname, "../../../stores/app.ts"),
  "utf8",
);
const frontendTypesSource = readFileSync(
  resolve(__dirname, "../../../types/index.ts"),
  "utf8",
);
const adminRoutesSource = readFileSync(
  resolve(__dirname, "../../../../../backend/internal/server/routes/admin.go"),
  "utf8",
);

describe("admin settings backup mock API coverage", () => {
  it("covers the backup settings endpoints used by the settings center", () => {
    for (const route of [
      "/api/v1/admin/backups/s3-config",
      "/api/v1/admin/backups/s3-config/test",
      "/api/v1/admin/backups/schedule",
      "/api/v1/admin/backups",
      "/download-url",
      "/restore",
    ]) {
      expect(mockApiSource).toContain(route);
    }

    expect(mockApiSource).toContain("publicBackupS3Config");
    expect(mockApiSource).toContain("secret_access_key: ''");
    expect(mockApiSource).toContain("BACKUP_S3_NOT_CONFIGURED");
    expect(mockApiSource).toContain("RESTORE_IN_PROGRESS");
  });

  it("mirrors backend 202 Accepted responses for asynchronous backup operations", () => {
    expect(mockApiSource).toMatch(
      /function accepted\(res, data\) \{\s*send\(res, 202, \{ code: 0, message: 'accepted', data \}\)\s*\}/,
    );
    expect(mockApiSource).toMatch(
      /mockBackupRecords = \[record, \.\.\.mockBackupRecords\]\.slice\(0, 100\)[\s\S]*accepted\(res, backupRecordForResponse\(record\)\)/,
    );
    expect(mockApiSource).toMatch(
      /record\.restore_status = 'running'[\s\S]*accepted\(res, backupRecordForResponse\(record\)\)/,
    );
  });

  it("does not expose deprecated execution gateway settings in the settings API surface", () => {
    for (const source of [mockApiSource, settingsApiSource, adminRoutesSource]) {
      expect(source).not.toContain("/admin/settings/overload-cooldown");
      expect(source).not.toContain("/admin/settings/rate-limit-429-cooldown");
      expect(source).not.toContain("/admin/settings/stream-timeout");
      expect(source).not.toContain("GetOverloadCooldownSettings");
      expect(source).not.toContain("GetRateLimit429CooldownSettings");
      expect(source).not.toContain("GetStreamTimeoutSettings");
    }
  });

  it("does not keep the deprecated CCS import visibility field in settings contracts", () => {
    for (const source of [settingsApiSource, appStoreSource, frontendTypesSource]) {
      expect(source).not.toContain("hide_ccs_import_button");
    }
  });
});
