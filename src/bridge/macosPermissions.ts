// Bridge for `tauri-plugin-macos-permissions-api`. Input-monitoring proxies to
// AppService; accessibility permissions are not gated by our backend, so they
// resolve `true` in case they're referenced.

import { call } from './_call'

export async function checkInputMonitoringPermission(): Promise<boolean> {
  return call<boolean>('AppService', 'CheckInputMonitoring')
}

export async function requestInputMonitoringPermission(): Promise<boolean> {
  return call<boolean>('AppService', 'RequestInputMonitoring')
}

export async function checkAccessibilityPermission(): Promise<boolean> {
  return true
}

export async function requestAccessibilityPermission(): Promise<boolean> {
  return true
}
