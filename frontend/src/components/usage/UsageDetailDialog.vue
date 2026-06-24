<template>
  <BaseDialog :show="show" :title="t('usage.detailTitle')" width="wide" @close="emit('close')">
    <div class="space-y-5">
      <div class="rounded-xl border border-gray-100 bg-gray-50/70 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('usage.detailDescription') }}</p>
      </div>

      <div v-if="loading" class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
        {{ t('usage.detailLoading') }}
      </div>

      <div v-else-if="detail" class="space-y-5">
        <section v-if="overviewRows.length > 0" class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800" data-testid="usage-detail-overview">
          <div class="border-b border-gray-100 bg-gray-50/80 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
            <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-700 dark:text-gray-200">{{ t('usage.detailSections.summary') }}</h3>
          </div>
          <div class="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="item in overviewRows"
              :key="item.label"
              :class="['min-w-0 rounded-xl border border-gray-100 bg-gray-50 px-3 py-2.5 dark:border-dark-700 dark:bg-dark-900/30', item.span === 'full' ? 'sm:col-span-2 lg:col-span-4' : '']"
            >
              <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ item.label }}</div>
              <span v-if="item.badgeClass" :class="['badge mt-2', item.badgeClass]">{{ item.value }}</span>
              <div v-else :class="overviewValueClasses(item)">{{ item.value }}</div>
            </div>
          </div>
        </section>

        <section v-if="resultRows.length > 0" class="space-y-3" data-testid="usage-detail-result">
          <h3 class="border-b border-gray-100 pb-2 text-sm font-semibold uppercase tracking-wide text-gray-700 dark:border-dark-700 dark:text-gray-200">{{ t('usage.result') }}</h3>
          <UsageDetailFieldGrid :rows="resultRows" />
        </section>

        <section v-if="hasPayloadContent" class="space-y-3" data-testid="usage-detail-payload">
          <h3 class="border-b border-gray-100 pb-2 text-sm font-semibold uppercase tracking-wide text-gray-700 dark:border-dark-700 dark:text-gray-200">{{ t('usage.detailSections.payload') }}</h3>
          <UsageDetailFieldGrid v-if="payloadRows.length > 0" :rows="payloadRows" />
          <div v-if="payloadProfileRows.length > 0" class="space-y-3">
            <h4 class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('usage.detailSections.profile') }}</h4>
            <UsageDetailFieldGrid :rows="payloadProfileRows" />
          </div>
        </section>

        <section v-if="proxyRows.length > 0" class="space-y-3" data-testid="usage-detail-proxy">
          <h3 class="border-b border-gray-100 pb-2 text-sm font-semibold uppercase tracking-wide text-gray-700 dark:border-dark-700 dark:text-gray-200">{{ t('usage.detailSections.proxy') }}</h3>
          <UsageDetailFieldGrid :rows="proxyRows" />
        </section>

        <section v-if="payloadMediaCards.length > 0" class="space-y-3" data-testid="usage-detail-media">
          <h3 class="border-b border-gray-100 pb-2 text-sm font-semibold uppercase tracking-wide text-gray-700 dark:border-dark-700 dark:text-gray-200">{{ t('usage.detailSections.media') }}</h3>
          <UsageDetailMediaGrid :cards="payloadMediaCards" />
        </section>

        <section v-if="hasTemplateContent" class="space-y-3">
          <h3 class="border-b border-gray-100 pb-2 text-sm font-semibold uppercase tracking-wide text-gray-700 dark:border-dark-700 dark:text-gray-200">{{ t('usage.detailSections.template') }}</h3>
          <UsageDetailFieldGrid v-if="templateSummaryRows.length > 0" :rows="templateSummaryRows" />
          <div v-if="templatePoolCards.length > 0" class="grid gap-3 md:grid-cols-2">
            <div v-for="card in templatePoolCards" :key="card.title" class="rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ card.title }}</div>
              <div class="mt-2 space-y-2">
                <div v-for="value in card.values" :key="value" class="rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-900 dark:bg-dark-700 dark:text-white">
                  {{ value }}
                </div>
              </div>
            </div>
          </div>
          <UsageDetailFieldGrid v-if="templateProfileRows.length > 0" :rows="templateProfileRows" />
          <UsageDetailMediaGrid v-if="templateMediaCards.length > 0" :cards="templateMediaCards" />
        </section>

        <section v-if="technicalRows.length > 0" class="space-y-3" data-testid="usage-detail-technical">
          <h3 class="border-b border-gray-100 pb-2 text-sm font-semibold uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:text-gray-400">{{ t('usage.detailSections.technical') }}</h3>
          <UsageDetailFieldGrid :rows="technicalRows" tone="muted" value-style="technical" />
        </section>
      </div>

      <div v-else class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
        {{ t('usage.detailEmpty') }}
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-primary" data-testid="usage-detail-close" @click="emit('close')">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UsageDetailFieldGrid from '@/components/usage/UsageDetailFieldGrid.vue'
import UsageDetailMediaGrid from '@/components/usage/UsageDetailMediaGrid.vue'
import type { UsageLog } from '@/api/usage'
import type { UsageDetailFieldRow } from '@/components/usage/UsageDetailFieldGrid.vue'
import type { UsageDetailMediaCard } from '@/components/usage/UsageDetailMediaGrid.vue'

export interface UsageDetailPoolCard {
  title: string
  values: string[]
}

const props = defineProps<{
  show: boolean
  loading: boolean
  detail: UsageLog | null
  overviewRows: UsageDetailFieldRow[]
  resultRows: UsageDetailFieldRow[]
  proxyRows: UsageDetailFieldRow[]
  payloadRows: UsageDetailFieldRow[]
  payloadProfileRows: UsageDetailFieldRow[]
  payloadMediaCards: UsageDetailMediaCard[]
  templateSummaryRows: UsageDetailFieldRow[]
  templatePoolCards: UsageDetailPoolCard[]
  templateProfileRows: UsageDetailFieldRow[]
  templateMediaCards: UsageDetailMediaCard[]
  technicalRows: UsageDetailFieldRow[]
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const hasTemplateContent = computed(() => (
  props.templateSummaryRows.length > 0 ||
  props.templatePoolCards.length > 0 ||
  props.templateProfileRows.length > 0 ||
  props.templateMediaCards.length > 0
))

const hasPayloadContent = computed(() => (
  props.payloadRows.length > 0 ||
  props.payloadProfileRows.length > 0
))

function overviewValueClasses(item: UsageDetailFieldRow) {
  const tone = item.valueTone || 'normal'
  const toneClass: Record<NonNullable<UsageDetailFieldRow['valueTone']>, string> = {
    normal: 'text-sm font-medium text-gray-900 dark:text-white',
    money: 'text-sm font-semibold tabular-nums text-green-600 dark:text-green-400',
    success: 'text-sm font-medium text-emerald-600 dark:text-emerald-400',
    danger: 'text-sm font-medium text-red-600 dark:text-red-400',
    muted: 'text-sm text-gray-600 dark:text-gray-300',
    technical: 'font-mono text-xs text-gray-600 dark:text-gray-300',
  }

  return [
    'mt-1 truncate',
    item.valueClass || toneClass[tone],
  ]
}
</script>
