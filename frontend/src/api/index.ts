/**
 * API Client for SocialOps Backend
 * Central export point for all API modules
 */

// Re-export the HTTP client
export { apiClient } from './client'

// Auth API
export { authAPI, isTotp2FARequired, type LoginResponse } from './auth'

// User APIs
export { userAPI } from './user'
export { usageAPI } from './usage'
export { redeemAPI, type RedeemHistoryItem } from './redeem'
export { paymentAPI } from './payment'
export { totpAPI } from './totp'
export { default as announcementsAPI } from './announcements'

// Admin APIs
export { adminAPI } from './admin'

// Default export
export { default } from './client'
