<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('admin.dataManagement.title') }}
        </h1>
        <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.dataManagement.description') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-secondary btn-sm w-full sm:w-auto"
        :disabled="loadingHealth"
        @click="loadAgentHealth"
      >
        {{ loadingHealth ? t('common.loading') : t('admin.dataManagement.actions.refresh') }}
      </button>
    </div>

    <section class="card p-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.dataManagement.agent.title') }}
          </h2>
          <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.dataManagement.agent.description') }}
          </p>
        </div>
        <span
          class="inline-flex w-fit items-center rounded-full px-3 py-1 text-xs font-medium"
          :class="agentBadgeClass"
        >
          {{ agentHealth?.enabled ? t('common.enabled') : t('common.disabled') }}
        </span>
      </div>

      <div v-if="loadingHealth && !agentHealth" class="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div v-for="idx in 4" :key="idx" class="h-16 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-700"></div>
      </div>

      <div v-else class="mt-5 space-y-4">
        <div v-if="healthError" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
          {{ healthError }}
        </div>

        <div v-if="agentHealth" class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <p class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.agent.status') }}
            </p>
            <p class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ agentHealth.enabled ? t('admin.dataManagement.agent.enabled') : t('admin.dataManagement.agent.disabled') }}
            </p>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <p class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.agent.socketPath') }}
            </p>
            <p class="mt-2 break-all font-mono text-xs text-gray-800 dark:text-gray-200">
              {{ agentHealth.socket_path || t('common.notAvailable') }}
            </p>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <p class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.agent.version') }}
            </p>
            <p class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ agentHealth.agent?.version || t('common.notAvailable') }}
            </p>
          </div>
          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <p class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.agent.uptime') }}
            </p>
            <p class="mt-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ formatUptime(agentHealth.agent?.uptime_seconds) }}
            </p>
          </div>
        </div>

        <div
          v-if="agentHealth && !agentHealth.enabled"
          class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200"
        >
          <p class="font-medium">{{ t('admin.dataManagement.agent.disabled') }}</p>
          <dl class="mt-3 grid gap-2 sm:grid-cols-[12rem_minmax(0,1fr)]">
            <dt class="text-amber-700 dark:text-amber-300">
              {{ t('admin.dataManagement.agent.reasonLabel') }}
            </dt>
            <dd>{{ resolvedAgentReason }}</dd>
            <dt class="text-amber-700 dark:text-amber-300">
              {{ t('admin.dataManagement.agent.socketPath') }}
            </dt>
            <dd class="break-all font-mono text-xs">{{ agentHealth.socket_path || t('common.notAvailable') }}</dd>
          </dl>
          <p class="mt-3 text-xs text-amber-700 dark:text-amber-300">
            {{ t('admin.dataManagement.actions.disabledHint') }}
          </p>
        </div>
      </div>
    </section>

    <template v-if="agentHealth?.enabled">
      <div v-if="dataError" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
        {{ dataError }}
      </div>

      <section class="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div class="card p-6">
          <div class="mb-4 flex items-start justify-between gap-3">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('admin.dataManagement.sections.config.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.dataManagement.sections.config.description') }}
              </p>
            </div>
            <button type="button" class="btn btn-secondary btn-xs" :disabled="loadingData" @click="loadAgentBackedData">
              {{ t('admin.dataManagement.actions.reloadConfig') }}
            </button>
          </div>

          <dl v-if="config" class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dataManagement.form.sourceMode') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ config.source_mode }}</dd>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dataManagement.form.retentionDays') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ config.retention_days }}</dd>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60 sm:col-span-2">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dataManagement.form.backupRoot') }}</dt>
              <dd class="mt-1 break-all font-mono text-xs text-gray-900 dark:text-white">{{ config.backup_root }}</dd>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dataManagement.form.activePostgresProfile') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ findProfileName(postgresProfiles, config.active_postgres_profile_id) }}</dd>
            </div>
            <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60">
              <dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.dataManagement.form.activeRedisProfile') }}</dt>
              <dd class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ findProfileName(redisProfiles, config.active_redis_profile_id) }}</dd>
            </div>
          </dl>
          <p v-else class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ loadingData ? t('common.loading') : t('common.noData') }}
          </p>
        </div>

        <div class="card p-6">
          <div class="mb-4">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dataManagement.sections.s3.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.sections.s3.description') }}
            </p>
          </div>

          <div v-if="s3Profiles.length" class="space-y-3">
            <div
              v-for="profile in s3Profiles"
              :key="profile.profile_id"
              class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
            >
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ profile.name }}</p>
                  <p class="mt-1 break-all font-mono text-xs text-gray-500 dark:text-gray-400">{{ profile.profile_id }}</p>
                </div>
                <span class="rounded-full px-2.5 py-1 text-xs" :class="profile.is_active ? activePillClass : mutedPillClass">
                  {{ profile.is_active ? t('common.active') : t('common.inactive') }}
                </span>
              </div>
              <p class="mt-3 break-all text-xs text-gray-600 dark:text-gray-300">
                {{ profile.s3.bucket || t('common.notAvailable') }} / {{ profile.s3.prefix || '-' }}
              </p>
            </div>
          </div>
          <p v-else class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ loadingData ? t('common.loading') : t('admin.dataManagement.s3Profiles.empty') }}
          </p>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <div class="card p-6">
          <div class="mb-4">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dataManagement.sourceProfiles.columns.profile') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.actions.reloadSourceProfiles') }}
            </p>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <h3 class="mb-2 text-sm font-semibold text-gray-800 dark:text-gray-100">
                {{ t('admin.dataManagement.form.postgres.title') }}
              </h3>
              <div v-if="postgresProfiles.length" class="space-y-2">
                <div v-for="profile in postgresProfiles" :key="profile.profile_id" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60">
                  <div class="flex items-center justify-between gap-2">
                    <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ profile.name }}</p>
                    <span class="rounded-full px-2 py-0.5 text-xs" :class="profile.is_active ? activePillClass : mutedPillClass">
                      {{ profile.is_active ? t('common.active') : t('common.inactive') }}
                    </span>
                  </div>
                  <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ formatSourceConnection(profile) }}</p>
                </div>
              </div>
              <p v-else class="rounded-lg bg-gray-50 p-4 text-center text-sm text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
                {{ t('admin.dataManagement.sourceProfiles.empty') }}
              </p>
            </div>

            <div>
              <h3 class="mb-2 text-sm font-semibold text-gray-800 dark:text-gray-100">
                {{ t('admin.dataManagement.form.redis.title') }}
              </h3>
              <div v-if="redisProfiles.length" class="space-y-2">
                <div v-for="profile in redisProfiles" :key="profile.profile_id" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/60">
                  <div class="flex items-center justify-between gap-2">
                    <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ profile.name }}</p>
                    <span class="rounded-full px-2 py-0.5 text-xs" :class="profile.is_active ? activePillClass : mutedPillClass">
                      {{ profile.is_active ? t('common.active') : t('common.inactive') }}
                    </span>
                  </div>
                  <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ formatSourceConnection(profile) }}</p>
                </div>
              </div>
              <p v-else class="rounded-lg bg-gray-50 p-4 text-center text-sm text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
                {{ t('admin.dataManagement.sourceProfiles.empty') }}
              </p>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <div class="mb-4">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dataManagement.sections.backup.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.sections.backup.description') }}
            </p>
          </div>

          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.dataManagement.history.columns.type') }}
              </label>
              <select v-model="backupForm.backup_type" class="input w-full">
                <option v-for="option in backupTypeOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.dataManagement.form.idempotencyKey') }}
              </label>
              <input v-model.trim="backupForm.idempotency_key" class="input w-full" />
            </div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 sm:col-span-2">
              <input v-model="backupForm.upload_to_s3" type="checkbox" />
              <span>{{ t('admin.dataManagement.form.uploadToS3') }}</span>
            </label>
          </div>

          <div class="mt-4 flex flex-wrap gap-2">
            <button type="button" class="btn btn-primary btn-sm" :disabled="creatingBackup || loadingData" @click="createBackupJob">
              {{ creatingBackup ? t('common.loading') : t('admin.dataManagement.actions.createBackup') }}
            </button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loadingData" @click="loadBackupJobs">
              {{ t('admin.dataManagement.actions.refreshJobs') }}
            </button>
          </div>
        </div>
      </section>

      <section class="card p-6">
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dataManagement.sections.history.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.dataManagement.sections.history.description') }}
            </p>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.dataManagement.history.total', { count: backupJobs.length }) }}
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full min-w-[900px] text-sm">
            <thead>
              <tr class="border-b border-gray-200 text-left text-xs uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:text-gray-400">
                <th class="py-2 pr-4">{{ t('admin.dataManagement.history.columns.jobID') }}</th>
                <th class="py-2 pr-4">{{ t('admin.dataManagement.history.columns.type') }}</th>
                <th class="py-2 pr-4">{{ t('admin.dataManagement.history.columns.status') }}</th>
                <th class="py-2 pr-4">{{ t('admin.dataManagement.history.columns.triggeredBy') }}</th>
                <th class="py-2 pr-4">{{ t('admin.dataManagement.history.columns.finishedAt') }}</th>
                <th class="py-2 pr-4">{{ t('admin.dataManagement.history.columns.artifact') }}</th>
                <th class="py-2">{{ t('admin.dataManagement.history.columns.error') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="job in backupJobs" :key="job.job_id" class="border-b border-gray-100 align-top dark:border-dark-800">
                <td class="py-3 pr-4 font-mono text-xs">{{ job.job_id }}</td>
                <td class="py-3 pr-4">{{ formatBackupType(job.backup_type) }}</td>
                <td class="py-3 pr-4">
                  <span class="rounded px-2 py-0.5 text-xs" :class="jobStatusClass(job.status)">
                    {{ formatJobStatus(job.status) }}
                  </span>
                </td>
                <td class="py-3 pr-4 text-xs">{{ job.triggered_by || '-' }}</td>
                <td class="py-3 pr-4 text-xs">{{ formatDate(job.finished_at || job.started_at) }}</td>
                <td class="py-3 pr-4 text-xs">{{ formatArtifact(job) }}</td>
                <td class="py-3 text-xs text-red-600 dark:text-red-300">{{ job.error_message || '-' }}</td>
              </tr>
              <tr v-if="backupJobs.length === 0">
                <td colspan="7" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ loadingData ? t('common.loading') : t('admin.dataManagement.history.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <section v-else-if="agentHealth && !agentHealth.enabled" class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div
        v-for="section in disabledSections"
        :key="section"
        class="rounded-lg border border-gray-200 bg-white p-5 opacity-80 dark:border-dark-700 dark:bg-dark-800"
      >
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t(`admin.dataManagement.sections.${section}.title`) }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t(`admin.dataManagement.sections.${section}.description`) }}
        </p>
        <p class="mt-4 rounded-lg bg-gray-50 p-3 text-xs text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
          {{ t('admin.dataManagement.actions.disabledHint') }}
        </p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { dataManagementAPI } from '@/api/admin/dataManagement'
import { useAppStore } from '@/stores'
import type {
  BackupAgentHealth,
  BackupJob,
  BackupJobStatus,
  BackupType,
  DataManagementConfig,
  DataManagementS3Profile,
  DataManagementSourceProfile
} from '@/api/admin/dataManagement'

const { t, te } = useI18n()
const appStore = useAppStore()

const agentHealth = ref<BackupAgentHealth | null>(null)
const loadingHealth = ref(false)
const healthError = ref('')
const loadingData = ref(false)
const dataError = ref('')

const config = ref<DataManagementConfig | null>(null)
const postgresProfiles = ref<DataManagementSourceProfile[]>([])
const redisProfiles = ref<DataManagementSourceProfile[]>([])
const s3Profiles = ref<DataManagementS3Profile[]>([])
const backupJobs = ref<BackupJob[]>([])
const creatingBackup = ref(false)

const backupForm = ref({
  backup_type: 'full' as BackupType,
  upload_to_s3: false,
  idempotency_key: ''
})

const disabledSections = ['config', 's3', 'backup', 'history'] as const
const activePillClass = 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
const mutedPillClass = 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'

const agentBadgeClass = computed(() => {
  if (agentHealth.value?.enabled) return activePillClass
  if (loadingHealth.value) return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200'
})

const resolvedAgentReason = computed(() => {
  const reason = agentHealth.value?.reason || 'UNKNOWN'
  const key = `admin.dataManagement.agent.reason.${reason}`
  return te(key) ? t(key) : t('admin.dataManagement.agent.reason.UNKNOWN')
})

const backupTypeOptions = computed(() => [
  { value: 'full' as BackupType, label: t('admin.dataManagement.history.types.full') },
  { value: 'postgres' as BackupType, label: t('admin.dataManagement.history.types.postgres') },
  { value: 'redis' as BackupType, label: t('admin.dataManagement.history.types.redis') }
])

function getErrorMessage(error: unknown): string {
  const message = (error as { message?: string })?.message
  return typeof message === 'string' && message.trim() ? message : t('errors.networkError')
}

function clearAgentBackedData() {
  config.value = null
  postgresProfiles.value = []
  redisProfiles.value = []
  s3Profiles.value = []
  backupJobs.value = []
  dataError.value = ''
}

async function loadAgentHealth() {
  loadingHealth.value = true
  healthError.value = ''
  try {
    agentHealth.value = await dataManagementAPI.getAgentHealth()
    if (!agentHealth.value?.enabled) {
      clearAgentBackedData()
      return
    }
    await loadAgentBackedData()
  } catch (error) {
    healthError.value = getErrorMessage(error)
    clearAgentBackedData()
    appStore.showError(healthError.value)
  } finally {
    loadingHealth.value = false
  }
}

async function loadAgentBackedData() {
  if (!agentHealth.value?.enabled) return
  loadingData.value = true
  dataError.value = ''
  try {
    const [nextConfig, nextPostgresProfiles, nextRedisProfiles, nextS3Profiles, nextBackupJobs] = await Promise.all([
      dataManagementAPI.getConfig(),
      dataManagementAPI.listSourceProfiles('postgres'),
      dataManagementAPI.listSourceProfiles('redis'),
      dataManagementAPI.listS3Profiles(),
      dataManagementAPI.listBackupJobs({ page_size: 10 })
    ])

    config.value = nextConfig
    postgresProfiles.value = nextPostgresProfiles.items || []
    redisProfiles.value = nextRedisProfiles.items || []
    s3Profiles.value = nextS3Profiles.items || []
    backupJobs.value = nextBackupJobs.items || []
  } catch (error) {
    dataError.value = getErrorMessage(error)
    appStore.showError(dataError.value)
  } finally {
    loadingData.value = false
  }
}

async function loadBackupJobs() {
  if (!agentHealth.value?.enabled) {
    appStore.showWarning(t('admin.dataManagement.actions.disabledHint'))
    return
  }
  loadingData.value = true
  try {
    const result = await dataManagementAPI.listBackupJobs({ page_size: 10 })
    backupJobs.value = result.items || []
  } catch (error) {
    const message = getErrorMessage(error)
    dataError.value = message
    appStore.showError(message)
  } finally {
    loadingData.value = false
  }
}

async function createBackupJob() {
  if (!agentHealth.value?.enabled) {
    appStore.showWarning(t('admin.dataManagement.actions.disabledHint'))
    return
  }

  creatingBackup.value = true
  try {
    const idempotencyKey = backupForm.value.idempotency_key.trim()
    const result = await dataManagementAPI.createBackupJob({
      backup_type: backupForm.value.backup_type,
      upload_to_s3: backupForm.value.upload_to_s3,
      idempotency_key: idempotencyKey || undefined
    })
    appStore.showSuccess(t('admin.dataManagement.actions.jobCreated', {
      jobID: result.job_id,
      status: formatJobStatus(result.status)
    }))
    await loadBackupJobs()
  } catch (error) {
    appStore.showError(getErrorMessage(error))
  } finally {
    creatingBackup.value = false
  }
}

function findProfileName(profiles: DataManagementSourceProfile[], profileID?: string): string {
  if (!profileID) return t('common.notAvailable')
  const profile = profiles.find((item) => item.profile_id === profileID)
  return profile ? profile.name : profileID
}

function formatSourceConnection(profile: DataManagementSourceProfile): string {
  if (profile.source_type === 'postgres') {
    const host = profile.config.host || profile.config.container_name || '-'
    return `${host}:${profile.config.port || '-'} / ${profile.config.database || '-'}`
  }
  return profile.config.addr || profile.config.container_name || '-'
}

function formatBackupType(type: BackupType | string): string {
  const key = `admin.dataManagement.history.types.${type}`
  return te(key) ? t(key) : String(type || '-')
}

function formatJobStatus(status: BackupJobStatus | string): string {
  const key = `admin.dataManagement.history.status.${status}`
  return te(key) ? t(key) : String(status || '-')
}

function jobStatusClass(status: BackupJobStatus): string {
  switch (status) {
    case 'succeeded':
      return activePillClass
    case 'running':
    case 'queued':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
    case 'partial_succeeded':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200'
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
    default:
      return mutedPillClass
  }
}

function formatUptime(seconds?: number): string {
  if (!seconds || seconds < 0) return t('common.notAvailable')
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatSize(bytes?: number): string {
  if (!bytes || bytes <= 0) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatArtifact(job: BackupJob): string {
  if (job.artifact?.size_bytes) return formatSize(job.artifact.size_bytes)
  if (job.s3?.key) return job.s3.key
  return '-'
}

onMounted(() => {
  loadAgentHealth()
})
</script>
