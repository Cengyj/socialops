export interface AdminAccountStub {
  id: number
  name: string
  status?: string
}

const accountsAPI = {
  async list(): Promise<{ items: AdminAccountStub[]; total: number; page: number; page_size: number; pages: number }> {
    return { items: [], total: 0, page: 1, page_size: 20, pages: 1 }
  },

  async getAll(): Promise<AdminAccountStub[]> {
    return []
  },
}

export default accountsAPI
