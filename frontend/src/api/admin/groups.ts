export interface AdminGroupStub {
  id: number
  name: string
  status?: string
}

const groupsAPI = {
  async getAll(): Promise<AdminGroupStub[]> {
    return []
  },

  async list(): Promise<{ items: AdminGroupStub[]; total: number; page: number; page_size: number; pages: number }> {
    return { items: [], total: 0, page: 1, page_size: 20, pages: 1 }
  },
}

export default groupsAPI
