<template>
  <div class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800/80">
    <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
      <div class="flex flex-1 flex-col gap-2 sm:flex-row sm:flex-wrap">
        <SearchInput
          :model-value="searchQuery"
          :placeholder="t('proxies.searchPlaceholder')"
          class="w-full sm:w-64"
          @update:model-value="emit('update:searchQuery', $event)"
        />
        <Select
          :model-value="statusFilter"
          :options="statusOptions"
          class="w-full sm:w-40"
          @update:model-value="emit('update:statusFilter', $event)"
        />
        <Select
          :model-value="typeFilter"
          :options="typeOptions"
          class="w-full sm:w-44"
          @update:model-value="emit('update:typeFilter', $event)"
        />
      </div>
      <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:flex xl:items-center">
        <button type="button" class="btn btn-secondary btn-sm h-10 justify-center" :disabled="loading" @click="emit('refresh')">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span>{{ t('common.refresh') }}</span>
        </button>
        <button type="button" class="btn btn-secondary btn-sm h-10 justify-center" :disabled="selectedCount === 0 || testing" @click="emit('testSelected')">
          <Icon name="play" size="sm" />
          <span>{{ t('proxies.testSelected') }}</span>
        </button>
        <button type="button" class="btn btn-secondary btn-sm h-10 justify-center" :disabled="testing || !hasProxies" @click="emit('testAll')">
          <Icon name="checkCircle" size="sm" />
          <span>{{ t('proxies.testAll') }}</span>
        </button>
        <button type="button" class="btn btn-primary btn-sm h-10 justify-center" @click="emit('create')">
          <Icon name="plus" size="sm" />
          <span>{{ t('proxies.addProxy') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { SelectOption } from '@/types'

type SelectValue = string | number | boolean | null

defineProps<{
  hasProxies: boolean
  loading: boolean
  searchQuery: string
  selectedCount: number
  statusFilter: SelectValue
  statusOptions: SelectOption[]
  testing: boolean
  typeFilter: SelectValue
  typeOptions: SelectOption[]
}>()

const emit = defineEmits<{
  create: []
  refresh: []
  testAll: []
  testSelected: []
  'update:searchQuery': [value: string]
  'update:statusFilter': [value: SelectValue]
  'update:typeFilter': [value: SelectValue]
}>()

const { t } = useI18n()
</script>
