export interface PageRef {
  name: string
  title: string
  summary: string
}

export interface Page {
  name: string
  title: string
  markdown: string
}

export class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = "ApiError"
  }
}

interface ListResponse {
  pages: PageRef[]
}

export async function listPages(baseURL: string): Promise<PageRef[]> {
  const res = await fetch(`${baseURL}/api/pages`)
  if (!res.ok) {
    throw new ApiError(`list pages: ${res.status}`)
  }
  const body = (await res.json()) as ListResponse
  return body.pages ?? []
}

export async function readPage(baseURL: string, name: string): Promise<Page> {
  const enc = encodeURIComponent(name)
  const res = await fetch(`${baseURL}/api/pages/${enc}`)
  if (!res.ok) {
    let msg = `read page: ${res.status}`
    try {
      const e = (await res.json()) as { error?: string }
      if (e?.error) msg = e.error
    } catch {
      // ignore
    }
    throw new ApiError(msg)
  }
  return (await res.json()) as Page
}