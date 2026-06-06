import type { AnnouncementOperator, AnnouncementTargeting } from '@/types'

export type AnnouncementTargetingValidationError =
  | 'too_many_or_groups'
  | 'empty_condition_group'
  | 'too_many_and_conditions'
  | 'invalid_condition_type'
  | 'invalid_subscription_operator'
  | 'subscription_group_required'
  | 'invalid_subscription_group_id'
  | 'invalid_balance_operator'

const maxConditionGroups = 50
const maxConditionsPerGroup = 50
const balanceOperators = new Set<AnnouncementOperator>(['gt', 'gte', 'lt', 'lte', 'eq'])

export function validateAnnouncementTargeting(
  targeting: AnnouncementTargeting | null | undefined
): AnnouncementTargetingValidationError | null {
  const anyOf = targeting?.any_of ?? []
  if (anyOf.length === 0) return null
  if (anyOf.length > maxConditionGroups) return 'too_many_or_groups'

  for (const group of anyOf) {
    const allOf = group?.all_of ?? []
    if (allOf.length === 0) return 'empty_condition_group'
    if (allOf.length > maxConditionsPerGroup) return 'too_many_and_conditions'

    for (const condition of allOf) {
      if (condition.type === 'subscription') {
        if (condition.operator !== 'in') return 'invalid_subscription_operator'
        const groupIDs = condition.group_ids ?? []
        if (groupIDs.length === 0) return 'subscription_group_required'
        if (groupIDs.some((id) => id <= 0)) return 'invalid_subscription_group_id'
        continue
      }

      if (condition.type === 'balance') {
        if (!balanceOperators.has(condition.operator)) return 'invalid_balance_operator'
        continue
      }

      return 'invalid_condition_type'
    }
  }

  return null
}

export function getAnnouncementTargetingValidationMessage(
  error: AnnouncementTargetingValidationError,
  translate: (key: string) => string
): string {
  switch (error) {
    case 'too_many_or_groups':
      return 'any_of > 50'
    case 'empty_condition_group':
      return translate('admin.announcements.form.addAndCondition')
    case 'too_many_and_conditions':
      return 'all_of > 50'
    case 'subscription_group_required':
    case 'invalid_subscription_group_id':
      return translate('admin.announcements.form.selectPackages')
    case 'invalid_condition_type':
    case 'invalid_subscription_operator':
    case 'invalid_balance_operator':
      return translate('admin.announcements.failedToCreate')
  }
}
