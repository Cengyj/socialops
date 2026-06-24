/**
 * Admin API barrel export
 * Centralized exports for all admin API modules
 */

import usersAPI from './users'
import redeemAPI from './redeem'
import promoAPI from './promo'
import announcementsAPI from './announcements'
import settingsAPI from './settings'
import systemAPI from './system'
import subscriptionsAPI from './subscriptions'
import userAttributesAPI from './userAttributes'
import backupAPI from './backup'
import adminPaymentAPI from './payment'
import affiliatesAPI from './affiliates'
import { accountWorkbenchAdminAPI } from '../accountWorkbench'
import dashboardAPI from './dashboard'
import groupsAPI from './groups'
import totalAccountsAPI from './totalAccounts'

/**
 * Unified admin API object for convenient access
 */
export const adminAPI = {
  users: usersAPI,
  redeem: redeemAPI,
  promo: promoAPI,
  announcements: announcementsAPI,
  settings: settingsAPI,
  system: systemAPI,
  subscriptions: subscriptionsAPI,
  userAttributes: userAttributesAPI,
  backup: backupAPI,
  payment: adminPaymentAPI,
  affiliates: affiliatesAPI,
  accountWorkbench: accountWorkbenchAdminAPI,
  dashboard: dashboardAPI,
  groups: groupsAPI,
  totalAccounts: totalAccountsAPI,
}

export {
  usersAPI,
  redeemAPI,
  promoAPI,
  announcementsAPI,
  settingsAPI,
  systemAPI,
  subscriptionsAPI,
  userAttributesAPI,
  backupAPI,
  adminPaymentAPI,
  affiliatesAPI,
  accountWorkbenchAdminAPI,
  dashboardAPI,
  groupsAPI,
  totalAccountsAPI,
}

export default adminAPI

// Re-export types used by components
export type { BalanceHistoryItem } from './users'
export type { AdminSocialTaskLog, SocialAccount, SocialAccountStats } from '../accountWorkbench'
