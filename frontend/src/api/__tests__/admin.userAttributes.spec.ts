import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { put } = vi.hoisted(() => ({
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    put,
  },
}))

import { updateUserAttributeValues } from '@/api/admin/userAttributes'
import type { UserAttributeValue } from '@/types'

const apiSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../admin/userAttributes.ts'),
  'utf8',
)
const updateUserAttributeValuesSource = apiSource.slice(
  apiSource.indexOf('export async function updateUserAttributeValues('),
  apiSource.indexOf('/**\n * Batch response type'),
)

describe('admin user attributes api', () => {
  beforeEach(() => {
    put.mockReset()
  })

  it('declares the user attribute value update response shape returned by the backend', () => {
    expect(apiSource).toContain(
      'export async function updateUserAttributeValues(\n' +
        '  userId: number,\n' +
        '  values: UserAttributeValuesMap\n' +
        '): Promise<UserAttributeValue[]>',
    )
    expect(updateUserAttributeValuesSource).not.toContain('): Promise<{ message: string }>')
  })

  it('unwraps updated user attribute values from the backend response', async () => {
    const response: UserAttributeValue[] = [{
      id: 11,
      user_id: 7,
      attribute_id: 3,
      value: 'gold',
      created_at: '2026-06-01T00:00:00Z',
      updated_at: '2026-06-01T00:00:00Z',
    }]
    put.mockResolvedValue({ data: response })

    const result = await updateUserAttributeValues(7, { 3: 'gold' })

    expect(put).toHaveBeenCalledWith('/admin/users/7/attributes', { values: { 3: 'gold' } })
    expect(result).toEqual(response)
  })
})
