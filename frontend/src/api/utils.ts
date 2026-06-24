export async function unwrapData<T>(request: Promise<{ data: T }>): Promise<T> {
  const { data } = await request
  return data
}
