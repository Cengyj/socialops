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
import dataManagementAPI from './dataManagement'
import backupAPI from './backup'
import adminPaymentAPI from './payment'
import affiliatesAPI from './affiliates'
import socialAccountsAdminAPI from './socialAccounts'
import adminUsageAPI from './usage'
import adminRiskControlAPI from './riskControl'
import dashboardAPI from './dashboard'
import groupsAPI from './groups'
import accountsAPI from './accounts'
import proxiesAPI from './proxies'

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
  dataManagement: dataManagementAPI,
  backup: backupAPI,
  payment: adminPaymentAPI,
  affiliates: affiliatesAPI,
  socialAccounts: socialAccountsAdminAPI,
  usage: adminUsageAPI,
  riskControl: adminRiskControlAPI,
  dashboard: dashboardAPI,
  groups: groupsAPI,
  accounts: accountsAPI,
  proxies: proxiesAPI,
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
  dataManagementAPI,
  backupAPI,
  adminPaymentAPI,
  affiliatesAPI,
  socialAccountsAdminAPI,
  adminUsageAPI,
  adminRiskControlAPI,
  dashboardAPI,
  groupsAPI,
  accountsAPI,
  proxiesAPI,
}

export default adminAPI

// Re-export types used by components
export type { BalanceHistoryItem } from './users'
export type { BackupAgentHealth, DataManagementConfig } from './dataManagement'
export type { SocialAccount, SocialAccountStats } from './socialAccounts'
export type { AdminProxy, AdminProxyCheckResult } from './proxies'
