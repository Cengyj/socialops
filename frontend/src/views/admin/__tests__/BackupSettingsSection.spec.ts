import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import BackupSettingsSection from '../settings/BackupSettingsSection.vue'

const {
  getS3Config,
  updateS3Config,
  testS3Connection,
  getSchedule,
  updateSchedule,
  createBackup,
  listBackups,
  getBackup,
  deleteBackup,
  getDownloadURL,
  restoreBackup,
  showError,
  showSuccess,
  showWarning,
} = vi.hoisted(() => ({
  getS3Config: vi.fn(),
  updateS3Config: vi.fn(),
  testS3Connection: vi.fn(),
  getSchedule: vi.fn(),
  updateSchedule: vi.fn(),
  createBackup: vi.fn(),
  listBackups: vi.fn(),
  getBackup: vi.fn(),
  deleteBackup: vi.fn(),
  getDownloadURL: vi.fn(),
  restoreBackup: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('@/api', () => ({
  adminAPI: {
    backup: {
      getS3Config,
      updateS3Config,
      testS3Connection,
      getSchedule,
      updateSchedule,
      createBackup,
      listBackups,
      getBackup,
      deleteBackup,
      getDownloadURL,
      restoreBackup,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const completedBackup = {
  id: 'backup-1',
  status: 'completed',
  backup_type: 'manual',
  file_name: 'socialops.sql.gz',
  s3_key: 'backups/socialops.sql.gz',
  size_bytes: 1024,
  triggered_by: 'manual',
  started_at: '2026-06-01T00:00:00Z',
  finished_at: '2026-06-01T00:00:02Z',
}

function mountSection() {
  return mount(BackupSettingsSection, {
    global: {
      stubs: {
        teleport: true,
        transition: false,
      },
    },
  })
}

describe('BackupSettingsSection', () => {
  beforeEach(() => {
    getS3Config.mockReset()
    updateS3Config.mockReset()
    testS3Connection.mockReset()
    getSchedule.mockReset()
    updateSchedule.mockReset()
    createBackup.mockReset()
    listBackups.mockReset()
    getBackup.mockReset()
    deleteBackup.mockReset()
    getDownloadURL.mockReset()
    restoreBackup.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()

    getS3Config.mockResolvedValue({
      endpoint: '',
      region: 'auto',
      bucket: '',
      access_key_id: '',
      prefix: 'backups/',
      force_path_style: false,
    })
    getSchedule.mockResolvedValue({
      enabled: false,
      cron_expr: '0 2 * * *',
      retain_days: 14,
      retain_count: 10,
    })
    listBackups.mockResolvedValue({ items: [] })
  })

  it('handles flattened 409 backup conflicts from apiClient as an in-progress warning', async () => {
    createBackup.mockRejectedValue({ status: 409, message: 'a backup is already in progress' })
    const wrapper = mountSection()
    await flushPromises()

    const button = wrapper
      .findAll('button')
      .find((node) => node.text().includes('admin.backup.operations.createBackup'))
    expect(button).toBeDefined()
    await button?.trigger('click')
    await flushPromises()

    expect(showWarning).toHaveBeenCalledWith('admin.backup.operations.alreadyInProgress')
    expect(showError).not.toHaveBeenCalledWith('a backup is already in progress')
  })

  it('handles flattened 409 restore conflicts from apiClient as a restore-running warning', async () => {
    listBackups.mockResolvedValue({ items: [completedBackup] })
    restoreBackup.mockRejectedValue({ status: 409, message: 'a restore is already in progress' })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.spyOn(window, 'prompt').mockReturnValue('admin-password')
    const wrapper = mountSection()
    await flushPromises()

    const button = wrapper
      .findAll('button')
      .find((node) => node.text().includes('admin.backup.actions.restore'))
    expect(button).toBeDefined()
    await button?.trigger('click')
    await flushPromises()

    expect(showWarning).toHaveBeenCalledWith('admin.backup.operations.restoreRunning')
    expect(showError).not.toHaveBeenCalledWith('a restore is already in progress')
  })

  it('preserves zero backup retention settings when saving an existing schedule', async () => {
    getSchedule.mockResolvedValue({
      enabled: true,
      cron_expr: '0 2 * * *',
      retain_days: 0,
      retain_count: 0,
    })
    const wrapper = mountSection()
    await flushPromises()

    const saveButtons = wrapper
      .findAll('button')
      .filter((node) => node.text().includes('common.save'))
    expect(saveButtons.length).toBeGreaterThan(1)
    await saveButtons[1].trigger('click')
    await flushPromises()

    expect(updateSchedule).toHaveBeenCalledWith({
      enabled: true,
      cron_expr: '0 2 * * *',
      retain_days: 0,
      retain_count: 0,
    })
  })
})
