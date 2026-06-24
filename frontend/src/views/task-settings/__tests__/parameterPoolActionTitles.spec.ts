import { describe, expect, it } from 'vitest'

import {
  parameterPoolClearButtonTitle,
  parameterPoolDedupeButtonTitle,
  parameterPoolImportButtonTitle,
  parameterPoolViewAllButtonTitle,
} from '../parameterPoolActionTitles'

const messages: Record<string, string> = {
  'common.saving': 'Saving',
  'taskSettings.clearValues': 'Clear values',
  'taskSettings.dedupe': 'Remove duplicates',
  'taskSettings.importFile': 'Import file',
  'taskSettings.pool.empty': 'No parameter values yet.',
  'taskSettings.pool.noDuplicates': 'There are no duplicate parameters to remove.',
  'taskSettings.viewAll': 'View all',
}

function t(key: string) {
  return messages[key] ?? key
}

describe('parameter pool action titles', () => {
  it('keeps idle toolbar action titles aligned with existing actions', () => {
    expect(parameterPoolImportButtonTitle(t, { saving: false })).toBe('Import file')
    expect(parameterPoolViewAllButtonTitle(t, { saving: false, valueCount: 2 })).toBe('View all')
    expect(parameterPoolDedupeButtonTitle(t, { saving: false, duplicateCount: 1 })).toBe('Remove duplicates')
    expect(parameterPoolClearButtonTitle(t, { saving: false, canClear: true })).toBe('Clear values')
  })

  it('explains unavailable pool actions without changing availability rules', () => {
    expect(parameterPoolViewAllButtonTitle(t, { saving: false, valueCount: 0 })).toBe('No parameter values yet.')
    expect(parameterPoolDedupeButtonTitle(t, { saving: false, duplicateCount: 0 })).toBe(
      'There are no duplicate parameters to remove.',
    )
    expect(parameterPoolClearButtonTitle(t, { saving: false, canClear: false })).toBe('No parameter values yet.')
  })

  it('prioritizes saving titles for actions locked during template operations', () => {
    expect(parameterPoolImportButtonTitle(t, { saving: true })).toBe('Saving')
    expect(parameterPoolViewAllButtonTitle(t, { saving: true, valueCount: 0 })).toBe('Saving')
    expect(parameterPoolDedupeButtonTitle(t, { saving: true, duplicateCount: 0 })).toBe('Saving')
    expect(parameterPoolClearButtonTitle(t, { saving: true, canClear: false })).toBe('Saving')
  })
})
