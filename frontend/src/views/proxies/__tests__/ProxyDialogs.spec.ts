import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { UserProxy } from '@/api/proxies'
import ProxyDialogs from '../ProxyDialogs.vue'

const { createProxy, updateProxy, deleteProxy, showError, showSuccess, recordClientDiagnostic } = vi.hoisted(() => ({
  createProxy: vi.fn(),
  updateProxy: vi.fn(),
  deleteProxy: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  recordClientDiagnostic: vi.fn(),
}))

vi.mock('@/api/proxies', () => ({
  default: {
    create: createProxy,
    update: updateProxy,
    delete: deleteProxy,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/utils/clientDiagnostics', () => ({
  recordClientDiagnostic,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.cancel': 'Cancel',
    'common.confirm': 'Confirm',
    'common.delete': 'Delete',
    'common.error': 'Something went wrong',
    'common.processing': 'Processing',
    'common.saving': 'Saving',
    'proxies.addProxy': 'Add proxy',
    'proxies.created': 'Proxy created',
    'proxies.deleteDialog.description': 'Delete {name}?',
    'proxies.deleteDialog.snapshotWarning': 'Account default proxy references will be cleared.',
    'proxies.deleteDialog.title': 'Delete proxy',
    'proxies.deleted': 'Proxy deleted',
    'proxies.deleteFailed': 'Failed to delete proxy',
    'proxies.editTitle': 'Edit proxy',
    'proxies.noChanges': 'No changes to save.',
    'proxies.form.endpoint': 'Endpoint',
    'proxies.form.endpointHint': 'Only online proxies with valid endpoints can be assigned.',
    'proxies.form.endpointPlaceholder': 'Proxy endpoint',
    'proxies.form.name': 'Name',
    'proxies.form.namePlaceholder': 'Proxy name',
    'proxies.form.remark': 'Remark',
    'proxies.form.remarkPlaceholder': 'Optional remark',
    'proxies.form.type': 'Type',
    'proxies.errors.INVALID_PROXY_ENDPOINT': 'Endpoint format is invalid.',
    'proxies.errors.SOCIAL_IP_SERVICE_UNAVAILABLE': 'Proxy service is temporarily unavailable.',
    'proxies.errors.SOCIAL_IP_INPUT_REQUIRED': 'Enter proxy information.',
    'proxies.errors.SOCIAL_IP_NAME_REQUIRED': 'Enter a proxy name.',
    'proxies.errors.SOCIAL_IP_NOT_FOUND': 'Proxy not found.',
    'proxies.errors.SOCIAL_IP_OWNER_NOT_FOUND': 'Signed-in account could not be verified.',
    'proxies.errors.SOCIAL_IP_TYPE_INVALID': 'Choose a valid proxy type.',
    'proxies.errors.SOCIAL_IP_USER_ID_NOT_ACCEPTED': 'Proxy ownership is fixed.',
    'proxies.saved': 'Proxy saved',
    'proxies.saveFailed': 'Failed to save proxy',
    'proxies.types.datacenter': 'Datacenter',
    'proxies.types.dynamic': 'Dynamic',
    'proxies.types.mobile': 'Mobile',
    'proxies.types.residential': 'Residential',
    'proxies.types.static': 'Static',
    'proxies.columns.endpoint': 'Endpoint',
    'proxies.columns.name': 'Name',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, paramsOrFallback?: Record<string, unknown> | string) => {
        const template = messages[key] ?? (typeof paramsOrFallback === 'string' ? paramsOrFallback : key)
        if (!paramsOrFallback || typeof paramsOrFallback === 'string') return template
        return Object.entries(paramsOrFallback).reduce(
          (text, [name, value]) => text.replace(`{${name}}`, String(value)),
          template,
        )
      },
    }),
  }
})

const baseProxy: UserProxy = {
  id: 7,
  user_id: 10,
  name: 'Tokyo proxy',
  ip_type: 'residential',
  endpoint: 'http://proxy.example.com:8080',
  status: 'unknown',
  latency_ms: null,
  last_check_at: null,
  remark: 'primary',
  created_at: '2026-06-06T00:00:00Z',
  updated_at: '2026-06-06T01:00:00Z',
}

function mountDialogs(props: Partial<InstanceType<typeof ProxyDialogs>['$props']> = {}) {
  return mount(ProxyDialogs, {
    props: {
      showForm: false,
      showDelete: false,
      editingProxy: null,
      proxyToDelete: null,
      ...props,
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<section v-if="show" role="dialog"><button type="button" aria-label="Close modal" @click="$emit(\'close\')"></button><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
        },
      },
    },
  })
}

function buttonByText(wrapper: ReturnType<typeof mountDialogs>, text: string) {
  const button = wrapper.findAll('button').find(item => item.text() === text)
  expect(button, `button ${text} should exist`).toBeTruthy()
  return button!
}

function expectConstrainedDialogButton(button: ReturnType<typeof buttonByText>, label: string) {
  expect(button.attributes('aria-label')).toBe(label)
  expect(button.attributes('title')).toBe(label)
  expect(button.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
  const text = button.findAll('span').find(item => item.text() === label)
  expect(text, `button text ${label} should be wrapped`).toBeTruthy()
  expect(text!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
}

async function clickDialogClose(wrapper: ReturnType<typeof mountDialogs>) {
  const close = wrapper.get('button[aria-label="Close modal"]')
  await close.trigger('click')
}

describe('ProxyDialogs', () => {
  beforeEach(() => {
    createProxy.mockReset()
    updateProxy.mockReset()
    deleteProxy.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    recordClientDiagnostic.mockReset()
  })

  it('keeps create submit disabled until the proxy name is present and supports cancel', async () => {
    const wrapper = mountDialogs({ showForm: true })

    const confirm = buttonByText(wrapper, 'Confirm')
    expect(confirm.attributes('disabled')).toBeDefined()
    expect(confirm.attributes('title')).toBe('Enter a proxy name.')
    await confirm.trigger('click')
    expect(createProxy).not.toHaveBeenCalled()

    await wrapper.find('#proxy-name').setValue('qa proxy')
    const readyConfirm = buttonByText(wrapper, 'Confirm')
    expect(readyConfirm.attributes('disabled')).toBeUndefined()
    expect(readyConfirm.attributes('title')).toBe('Confirm')

    await buttonByText(wrapper, 'Cancel').trigger('click')
    expect(wrapper.emitted('closeForm')).toHaveLength(1)
  })

  it('keeps the endpoint hint readable and inspectable in the proxy form', () => {
    const wrapper = mountDialogs({ showForm: true })
    const hint = wrapper.get('[title="Only online proxies with valid endpoints can be assigned."]')

    expect(hint.text()).toBe('Only online proxies with valid endpoints can be assigned.')
    expect(hint.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
  })

  it('keeps proxy dialog footer actions inspectable and constrained on narrow layouts', async () => {
    const createWrapper = mountDialogs({ showForm: true })

    expectConstrainedDialogButton(buttonByText(createWrapper, 'Cancel'), 'Cancel')
    const confirm = buttonByText(createWrapper, 'Confirm')
    expect(confirm.attributes('aria-label')).toBe('Confirm')
    expect(confirm.attributes('title')).toBe('Enter a proxy name.')
    expect(confirm.classes()).toEqual(expect.arrayContaining(['min-w-0', 'max-w-full', 'justify-center']))
    const confirmText = confirm.findAll('span').find(item => item.text() === 'Confirm')
    expect(confirmText, 'button text Confirm should be wrapped').toBeTruthy()
    expect(confirmText!.classes()).toEqual(expect.arrayContaining(['min-w-0', 'truncate']))
    expect(confirm.attributes('disabled')).toBeDefined()

    await createWrapper.find('#proxy-name').setValue('qa proxy')
    expect(buttonByText(createWrapper, 'Confirm').attributes('disabled')).toBeUndefined()
    expect(buttonByText(createWrapper, 'Confirm').attributes('title')).toBe('Confirm')

    const deleteWrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    expectConstrainedDialogButton(buttonByText(deleteWrapper, 'Cancel'), 'Cancel')
    const deleteButton = buttonByText(deleteWrapper, 'Delete')
    expectConstrainedDialogButton(deleteButton, 'Delete')
    expect(deleteButton.attributes('disabled')).toBeUndefined()
    const snapshotWarning = deleteWrapper.get('[title="Account default proxy references will be cleared."]')
    expect(snapshotWarning.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(snapshotWarning.attributes('role')).toBe('status')
    expect(snapshotWarning.attributes('aria-live')).toBe('polite')
    expect(snapshotWarning.attributes('aria-atomic')).toBe('true')
  })

  it('creates a proxy with trimmed delivery fields and emits the saved proxy', async () => {
    createProxy.mockResolvedValue(baseProxy)
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('  Tokyo proxy  ')
    await wrapper.find('#proxy-endpoint').setValue('  http://proxy.example.com:8080  ')
    await wrapper.find('#proxy-remark').setValue('  primary  ')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(createProxy).toHaveBeenCalledWith({
      name: 'Tokyo proxy',
      ip_type: 'residential',
      endpoint: 'http://proxy.example.com:8080',
      remark: 'primary',
    })
    expect(showSuccess).toHaveBeenCalledWith('Proxy created')
    expect(wrapper.emitted('saved')?.[0]).toEqual([baseProxy])
  })

  it('locks proxy form fields and actions while save is pending', async () => {
    let resolveCreate: ((proxy: UserProxy) => void) | undefined
    createProxy.mockImplementation(() => new Promise<UserProxy>(resolve => {
      resolveCreate = resolve
    }))
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    await wrapper.find('#proxy-endpoint').setValue('http://proxy.example.com:8080')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(createProxy).toHaveBeenCalledTimes(1)
    expect(wrapper.find<HTMLInputElement>('#proxy-name').element.disabled).toBe(true)
    expect(wrapper.find<HTMLSelectElement>('#proxy-type').element.disabled).toBe(true)
    expect(wrapper.find<HTMLInputElement>('#proxy-endpoint').element.disabled).toBe(true)
    expect(wrapper.find<HTMLTextAreaElement>('#proxy-remark').element.disabled).toBe(true)
    expect(buttonByText(wrapper, 'Cancel').attributes('disabled')).toBeDefined()
    expect(buttonByText(wrapper, 'Cancel').attributes('aria-label')).toBe('Saving')
    expect(buttonByText(wrapper, 'Cancel').attributes('title')).toBe('Saving')
    expect(buttonByText(wrapper, 'Saving').attributes('disabled')).toBeDefined()
    expect(buttonByText(wrapper, 'Saving').attributes('title')).toBe('Saving')

    await clickDialogClose(wrapper)
    await buttonByText(wrapper, 'Cancel').trigger('click')
    await buttonByText(wrapper, 'Saving').trigger('click')
    expect(wrapper.emitted('closeForm')).toBeUndefined()
    expect(createProxy).toHaveBeenCalledTimes(1)

    resolveCreate?.(baseProxy)
    await flushPromises()
    expect(wrapper.emitted('saved')?.[0]).toEqual([baseProxy])
  })

  it('keeps the form open and shows a safe error when proxy save fails', async () => {
    createProxy.mockRejectedValue(new Error('internal dial details'))
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.save', expect.any(Error))
    expect(showError).toHaveBeenCalledWith('Failed to save proxy')
    expect(showError).not.toHaveBeenCalledWith('Something went wrong')
    expect(wrapper.text()).toContain('Failed to save proxy')
    expect(wrapper.text()).not.toContain('internal dial details')
    expect(wrapper.find('#proxy-name').element).toHaveProperty('value', 'Tokyo proxy')
    const formError = wrapper.get('[title="Failed to save proxy"]')
    expect(formError.attributes('role')).toBe('alert')
    expect(formError.attributes('aria-live')).toBe('assertive')
    expect(formError.attributes('aria-atomic')).toBe('true')
  })

  it('maps known proxy save errors to friendly form messages before raw backend text', async () => {
    createProxy.mockRejectedValue({ code: 'INVALID_PROXY_ENDPOINT', message: 'invalid proxy endpoint URL' })
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    await wrapper.find('#proxy-endpoint').setValue('not a proxy endpoint')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.save', expect.objectContaining({ code: 'INVALID_PROXY_ENDPOINT' }))
    expect(showError).toHaveBeenCalledWith('Endpoint format is invalid.')
    expect(wrapper.text()).toContain('Endpoint format is invalid.')
    expect(wrapper.text()).not.toContain('invalid proxy endpoint URL')

    const formError = wrapper.get('[title="Endpoint format is invalid."]')
    expect(formError.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(formError.attributes('role')).toBe('alert')
    expect(formError.attributes('aria-live')).toBe('assertive')
    expect(formError.attributes('aria-atomic')).toBe('true')
  })

  it('maps proxy service availability save errors to the shared friendly message', async () => {
    createProxy.mockRejectedValue({ code: 'SOCIAL_IP_SERVICE_UNAVAILABLE', message: 'social IP service is unavailable' })
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.save', expect.objectContaining({ code: 'SOCIAL_IP_SERVICE_UNAVAILABLE' }))
    expect(showError).toHaveBeenCalledWith('Proxy service is temporarily unavailable.')
    expect(wrapper.text()).toContain('Proxy service is temporarily unavailable.')
    expect(wrapper.text()).not.toContain('social IP service is unavailable')
  })

  it('maps missing proxy owner save errors to a sign-in recovery message', async () => {
    createProxy.mockRejectedValue({ code: 'SOCIAL_IP_OWNER_NOT_FOUND', message: 'social IP owner not found' })
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.save', expect.objectContaining({ code: 'SOCIAL_IP_OWNER_NOT_FOUND' }))
    expect(showError).toHaveBeenCalledWith('Signed-in account could not be verified.')
    expect(wrapper.text()).toContain('Signed-in account could not be verified.')
    expect(wrapper.text()).not.toContain('social IP owner not found')
  })

  it('clears a stale save error when the user edits the proxy form again', async () => {
    createProxy.mockRejectedValue(new Error('internal dial details'))
    const wrapper = mountDialogs({ showForm: true })

    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to save proxy')

    await wrapper.find('#proxy-endpoint').setValue('http://proxy.example.com:8080')

    expect(wrapper.text()).not.toContain('Failed to save proxy')
    expect(createProxy).toHaveBeenCalledTimes(1)
  })

  it('updates an edited proxy through the current-user proxy endpoint', async () => {
    updateProxy.mockResolvedValue({ ...baseProxy, name: 'Tokyo proxy updated' })
    const wrapper = mountDialogs({
      showForm: true,
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    const initialConfirm = buttonByText(wrapper, 'Confirm')
    expect(initialConfirm.attributes('disabled')).toBeDefined()
    expect(initialConfirm.attributes('title')).toBe('No changes to save.')
    await initialConfirm.trigger('click')
    await flushPromises()
    expect(updateProxy).not.toHaveBeenCalled()

    await wrapper.find('#proxy-name').setValue('Tokyo proxy updated')
    await buttonByText(wrapper, 'Confirm').trigger('click')
    await flushPromises()

    expect(updateProxy).toHaveBeenCalledWith(7, {
      name: 'Tokyo proxy updated',
      ip_type: 'residential',
      endpoint: 'http://proxy.example.com:8080',
      remark: 'primary',
    })
    expect(createProxy).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('Proxy saved')
    expect(wrapper.emitted('saved')?.[0]?.[0]).toMatchObject({ id: 7, name: 'Tokyo proxy updated' })
  })

  it('keeps whitespace-only edit changes disabled because saved proxy fields are trimmed', async () => {
    updateProxy.mockResolvedValue(baseProxy)
    const wrapper = mountDialogs({
      showForm: true,
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await wrapper.find('#proxy-name').setValue('  Tokyo proxy  ')
    await wrapper.find('#proxy-endpoint').setValue('  http://proxy.example.com:8080  ')
    await wrapper.find('#proxy-remark').setValue('  primary  ')

    const unchangedConfirm = buttonByText(wrapper, 'Confirm')
    expect(unchangedConfirm.attributes('disabled')).toBeDefined()
    expect(unchangedConfirm.attributes('title')).toBe('No changes to save.')
    await unchangedConfirm.trigger('click')
    await flushPromises()

    expect(updateProxy).not.toHaveBeenCalled()

    await wrapper.find('#proxy-remark').setValue('  primary updated  ')
    const changedConfirm = buttonByText(wrapper, 'Confirm')
    expect(changedConfirm.attributes('disabled')).toBeUndefined()
    expect(changedConfirm.attributes('title')).toBe('Confirm')
    await changedConfirm.trigger('click')
    await flushPromises()

    expect(updateProxy).toHaveBeenCalledWith(7, {
      name: 'Tokyo proxy',
      ip_type: 'residential',
      endpoint: 'http://proxy.example.com:8080',
      remark: 'primary updated',
    })
  })

  it('locks an edited proxy form while the parent proxy list is busy', async () => {
    updateProxy.mockResolvedValue({ ...baseProxy, name: 'Tokyo proxy updated' })
    const wrapper = mountDialogs({
      showForm: true,
      formLocked: true,
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    expect(wrapper.find<HTMLInputElement>('#proxy-name').element.disabled).toBe(true)
    expect(wrapper.find<HTMLSelectElement>('#proxy-type').element.disabled).toBe(true)
    expect(wrapper.find<HTMLInputElement>('#proxy-endpoint').element.disabled).toBe(true)
    expect(wrapper.find<HTMLTextAreaElement>('#proxy-remark').element.disabled).toBe(true)

    const confirm = buttonByText(wrapper, 'Confirm')
    expect(confirm.attributes('disabled')).toBeDefined()
    expect(confirm.attributes('title')).toBe('Processing')
    await confirm.trigger('click')
    await flushPromises()

    expect(updateProxy).not.toHaveBeenCalled()
    expect(wrapper.emitted('saved')).toBeUndefined()

    await wrapper.setProps({ formLocked: false })
    expect(wrapper.find<HTMLInputElement>('#proxy-name').element.disabled).toBe(false)
    const unchangedConfirm = buttonByText(wrapper, 'Confirm')
    expect(unchangedConfirm.attributes('disabled')).toBeDefined()
    expect(unchangedConfirm.attributes('title')).toBe('No changes to save.')

    await wrapper.find('#proxy-remark').setValue('primary after refresh')
    const changedConfirm = buttonByText(wrapper, 'Confirm')
    expect(changedConfirm.attributes('disabled')).toBeUndefined()
    expect(changedConfirm.attributes('title')).toBe('Confirm')
  })

  it('refreshes a pristine edit form when the edited proxy prop updates', async () => {
    const wrapper = mountDialogs({
      showForm: true,
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://old-proxy.example.com:8080',
        remark: 'old note',
      },
    })

    expect((wrapper.find('#proxy-name').element as HTMLInputElement).value).toBe('Tokyo proxy')
    expect((wrapper.find('#proxy-endpoint').element as HTMLInputElement).value).toBe('http://old-proxy.example.com:8080')

    await wrapper.setProps({
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy updated',
        type: 'static',
        endpoint: 'http://new-proxy.example.com:8080',
        remark: 'new note',
      },
    })

    expect((wrapper.find('#proxy-name').element as HTMLInputElement).value).toBe('Tokyo proxy updated')
    expect((wrapper.find('#proxy-type').element as HTMLSelectElement).value).toBe('static')
    expect((wrapper.find('#proxy-endpoint').element as HTMLInputElement).value).toBe('http://new-proxy.example.com:8080')
    expect((wrapper.find('#proxy-remark').element as HTMLTextAreaElement).value).toBe('new note')
  })

  it('does not overwrite dirty edit fields when the edited proxy prop updates', async () => {
    const wrapper = mountDialogs({
      showForm: true,
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://old-proxy.example.com:8080',
        remark: 'old note',
      },
    })

    await wrapper.find('#proxy-name').setValue('Unsaved proxy name')
    await wrapper.find('#proxy-remark').setValue('unsaved note')

    await wrapper.setProps({
      editingProxy: {
        id: 7,
        name: 'Tokyo proxy updated',
        type: 'static',
        endpoint: 'http://new-proxy.example.com:8080',
        remark: 'new note',
      },
    })

    expect((wrapper.find('#proxy-name').element as HTMLInputElement).value).toBe('Unsaved proxy name')
    expect((wrapper.find('#proxy-type').element as HTMLSelectElement).value).toBe('residential')
    expect((wrapper.find('#proxy-endpoint').element as HTMLInputElement).value).toBe('http://old-proxy.example.com:8080')
    expect((wrapper.find('#proxy-remark').element as HTMLTextAreaElement).value).toBe('unsaved note')
  })

  it('keeps new proxy creation editable while the parent proxy list is refreshing', async () => {
    createProxy.mockResolvedValue(baseProxy)
    const wrapper = mountDialogs({
      showForm: true,
      formLocked: true,
      editingProxy: null,
    })

    expect(wrapper.find<HTMLInputElement>('#proxy-name').element.disabled).toBe(false)
    await wrapper.find('#proxy-name').setValue('Tokyo proxy')
    expect(buttonByText(wrapper, 'Confirm').attributes('disabled')).toBeUndefined()
  })

  it('deletes a proxy only after confirmation and supports cancel', async () => {
    deleteProxy.mockResolvedValue(undefined)
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await buttonByText(wrapper, 'Cancel').trigger('click')
    expect(wrapper.emitted('closeDelete')).toHaveLength(1)
    expect(deleteProxy).not.toHaveBeenCalled()

    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(deleteProxy).toHaveBeenCalledWith(7)
    expect(showSuccess).toHaveBeenCalledWith('Proxy deleted')
    expect(wrapper.emitted('deleted')?.[0]).toEqual([7])
  })

  it('keeps long proxy labels readable in the delete confirmation', () => {
    const longName = 'stage107_proxy_delete_name_with_really_long_unbroken_identifier_0123456789abcdef'
    const longEndpoint = 'http://stage107-proxy-endpoint-with-a-really-long-unbroken-hostname-0123456789abcdef.example.test:18080'
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: longName,
        type: 'residential',
        endpoint: longEndpoint,
        remark: 'primary',
      },
    })

    const deleteDescription = `Delete ${longName}?`
    const nameRow = wrapper.findAll('dd').find(node => node.text() === longName)
    const endpointRow = wrapper.findAll('dd').find(node => node.text() === longEndpoint)
    const description = wrapper.findAll('p').find(node => node.text() === deleteDescription)

    expect(description?.attributes('title')).toBe(deleteDescription)
    expect(description?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(nameRow?.attributes('title')).toBe(longName)
    expect(nameRow?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-all', 'sm:truncate']))
    expect(endpointRow?.attributes('title')).toBe(longEndpoint)
    expect(endpointRow?.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-all', 'sm:truncate']))
  })

  it('locks delete confirmation while the parent proxy list is busy', async () => {
    deleteProxy.mockResolvedValue(undefined)
    const wrapper = mountDialogs({
      showDelete: true,
      deleteLocked: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    const deleteButton = buttonByText(wrapper, 'Delete')
    expect(deleteButton.attributes('disabled')).toBeDefined()
    expect(deleteButton.attributes('title')).toBe('Processing')
    await deleteButton.trigger('click')
    await flushPromises()

    expect(deleteProxy).not.toHaveBeenCalled()
    expect(wrapper.emitted('deleted')).toBeUndefined()
    expect(wrapper.text()).toContain('Account default proxy references will be cleared.')

    await wrapper.setProps({ deleteLocked: false })
    expect(buttonByText(wrapper, 'Delete').attributes('disabled')).toBeUndefined()
  })

  it('keeps the delete confirmation open while delete is pending', async () => {
    let resolveDelete: (() => void) | undefined
    deleteProxy.mockImplementation(() => new Promise<void>((resolve) => {
      resolveDelete = resolve
    }))
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(deleteProxy).toHaveBeenCalledWith(7)
    expect(buttonByText(wrapper, 'Cancel').attributes('disabled')).toBeDefined()
    expect(buttonByText(wrapper, 'Cancel').attributes('aria-label')).toBe('Processing')
    expect(buttonByText(wrapper, 'Cancel').attributes('title')).toBe('Processing')
    expect(buttonByText(wrapper, 'Processing').attributes('disabled')).toBeDefined()
    expect(buttonByText(wrapper, 'Processing').attributes('title')).toBe('Processing')

    await clickDialogClose(wrapper)
    await buttonByText(wrapper, 'Cancel').trigger('click')
    await buttonByText(wrapper, 'Processing').trigger('click')

    expect(wrapper.emitted('closeDelete')).toBeUndefined()
    expect(deleteProxy).toHaveBeenCalledTimes(1)

    resolveDelete?.()
    await flushPromises()

    expect(showSuccess).toHaveBeenCalledWith('Proxy deleted')
    expect(wrapper.emitted('deleted')?.[0]).toEqual([7])
  })

  it('keeps the delete dialog open and shows a safe error when delete fails', async () => {
    deleteProxy.mockRejectedValue(new Error('sql constraint details'))
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('deleted')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.delete', expect.any(Error))
    expect(showError).toHaveBeenCalledWith('Failed to delete proxy')
    expect(wrapper.text()).toContain('Tokyo proxy')
    expect(wrapper.text()).toContain('Failed to delete proxy')
    const deleteError = wrapper.get('[title="Failed to delete proxy"]')
    expect(deleteError.classes()).toEqual(expect.arrayContaining(['min-w-0', 'break-words']))
    expect(deleteError.attributes('role')).toBe('alert')
    expect(deleteError.attributes('aria-live')).toBe('assertive')
    expect(deleteError.attributes('aria-atomic')).toBe('true')
  })

  it('clears a stale delete error when the delete target changes or closes', async () => {
    deleteProxy.mockRejectedValue(new Error('sql constraint details'))
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to delete proxy')

    await wrapper.setProps({
      proxyToDelete: {
        id: 8,
        name: 'Osaka proxy',
        type: 'static',
        endpoint: 'http://osaka-proxy.example.com:8080',
        remark: 'secondary',
      },
    })

    expect(wrapper.text()).not.toContain('Failed to delete proxy')

    deleteProxy.mockRejectedValue(new Error('sql constraint details'))
    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to delete proxy')

    await buttonByText(wrapper, 'Cancel').trigger('click')
    expect(wrapper.emitted('closeDelete')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('Failed to delete proxy')
  })

  it('maps known proxy delete errors to friendly messages before raw backend text', async () => {
    deleteProxy.mockRejectedValue({ code: 'SOCIAL_IP_NOT_FOUND', message: 'social IP not found' })
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('deleted')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.delete', expect.objectContaining({ code: 'SOCIAL_IP_NOT_FOUND' }))
    expect(showError).toHaveBeenCalledWith('Proxy not found.')
    expect(wrapper.text()).not.toContain('social IP not found')
  })

  it('maps proxy service availability delete errors to the shared friendly message', async () => {
    deleteProxy.mockRejectedValue({ code: 'SOCIAL_IP_SERVICE_UNAVAILABLE', message: 'social IP service is unavailable' })
    const wrapper = mountDialogs({
      showDelete: true,
      proxyToDelete: {
        id: 7,
        name: 'Tokyo proxy',
        type: 'residential',
        endpoint: 'http://proxy.example.com:8080',
        remark: 'primary',
      },
    })

    await buttonByText(wrapper, 'Delete').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('deleted')).toBeUndefined()
    expect(recordClientDiagnostic).toHaveBeenCalledWith('proxies.delete', expect.objectContaining({ code: 'SOCIAL_IP_SERVICE_UNAVAILABLE' }))
    expect(showError).toHaveBeenCalledWith('Proxy service is temporarily unavailable.')
    expect(wrapper.text()).not.toContain('social IP service is unavailable')
  })
})
