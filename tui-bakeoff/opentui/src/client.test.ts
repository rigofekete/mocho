import { test, expect, afterEach } from "bun:test"
import { listPages, readPage, ApiError } from "./client.js"

let server: ReturnType<typeof Bun.serve> | null = null

type Handler = (req: Request) => Response | Promise<Response>

function startServer(handler: Handler): string {
  stopServer()
  server = Bun.serve({ port: 0, fetch: handler })
  return `http://${server.hostname}:${server.port}`
}

afterEach(() => {
  stopServer()
  server = null
})

function stopServer() {
  if (server) server.stop(true)
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  })
}

test("listPages returns parsed pages", async () => {
  const base = startServer((req) => {
    if (req.method !== "GET" || new URL(req.url).pathname !== "/api/pages") {
      return json({ error: "unexpected" }, 400)
    }
    return json({
      pages: [
        { name: "concepts/goroutines.md", title: "Goroutines", summary: "lightweight threads" },
        { name: "courses/boot.md", title: "Boot", summary: "course hub" },
      ],
    })
  })

  const pages = await listPages(base)
  expect(pages).toHaveLength(2)
  expect(pages[0].name).toBe("concepts/goroutines.md")
  expect(pages[0].title).toBe("Goroutines")
  expect(pages[0].summary).toBe("lightweight threads")
})

test("listPages returns empty list cleanly", async () => {
  const base = startServer(() => json({ pages: [] }))
  const pages = await listPages(base)
  expect(pages).toEqual([])
})

test("listPages throws on server error", async () => {
  const base = startServer(() => json({ error: "boom" }, 500))
  await expect(listPages(base)).rejects.toBeInstanceOf(ApiError)
})

test("readPage sends encoded name and returns markdown", async () => {
  const base = startServer((req) => {
    const path = new URL(req.url).pathname
    expect(path).toBe("/api/pages/concepts%2Fgoroutines.md")
    return json({
      name: "concepts/goroutines.md",
      title: "Goroutines",
      markdown: "# Goroutines\n\nbody",
    })
  })

  const page = await readPage(base, "concepts/goroutines.md")
  expect(page.title).toBe("Goroutines")
  expect(page.markdown).toBe("# Goroutines\n\nbody")
})

test("readPage throws ApiError with server message on 404", async () => {
  const base = startServer(() => json({ error: "page not found" }, 404))
  await expect(readPage(base, "missing.md")).rejects.toThrow("page not found")
})