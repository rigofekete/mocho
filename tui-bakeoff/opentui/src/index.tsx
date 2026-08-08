import { createCliRenderer, SyntaxStyle, type ImageRenderProtocol } from "@opentui/core"
import { createRoot, useKeyboard } from "@opentui/react"
import { useEffect, useState } from "react"
import { Jimp, type JimpInstance } from "jimp"
import { listPages, readPage, type Page, type PageRef } from "./client.js"

const BASE_URL = process.env.MOCHO_API ?? "http://127.0.0.1:7777"

const COLOR_BG = "#0D1117"
const COLOR_FG = "#E6EDF3"
const COLOR_MUTED = "#6E7681"
const COLOR_ACCENT = "#58A6FF"
const COLOR_SELECTED = "#FF79C6"

const syntaxStyle = SyntaxStyle.fromStyles({
  keyword: { fg: "#FF7B72", bold: true },
  string: { fg: "#A5D6FF" },
  comment: { fg: "#8B949E", italic: true },
  number: { fg: "#79C0FF" },
  function: { fg: "#D2A8FF" },
  type: { fg: "#FFA657" },
  operator: { fg: "#FF7B72" },
  variable: { fg: "#E6EDF3" },
  property: { fg: "#79C0FF" },
  "punctuation.bracket": { fg: "#F0F6FC" },
  "punctuation.delimiter": { fg: "#C9D1D9" },
  "markup.heading": { fg: "#58A6FF", bold: true },
  "markup.heading.1": { fg: "#00FF88", bold: true, italic: true, underline: true },
  "markup.heading.2": { fg: "#00D7FF", bold: true },
  "markup.heading.3": { fg: "#FF69B4" },
  "markup.bold": { fg: "#F0F6FC", bold: true },
  "markup.strong": { fg: "#F0F6FC", bold: true },
  "markup.italic": { fg: "#F0F6FC", italic: true },
  "markup.list": { fg: "#FF7B72" },
  "markup.quote": { fg: "#8B949E", italic: true },
  "markup.raw": { fg: "#A5D6FF", bg: "#161B22" },
  "markup.raw.block": { fg: "#A5D6FF", bg: "#161B22" },
  "markup.raw.inline": { fg: "#A5D6FF", bg: "#161B22" },
  "markup.link": { fg: "#58A6FF", underline: true },
  "markup.link.label": { fg: "#A5D6FF", underline: true },
  "markup.link.url": { fg: "#58A6FF", underline: true },
  "markup.table": { fg: "#E6EDF3" },
  "diff.plus": { fg: "#3FB950" },
  "diff.minus": { fg: "#F85149" },
  label: { fg: "#7EE787" },
  conceal: { fg: "#6E7681" },
  "punctuation.special": { fg: "#8B949E" },
  default: { fg: "#E6EDF3" },
})

const FONT3X5: Record<string, number[]> = {
  " ": [0, 0, 0, 0, 0],
  A: [7, 5, 7, 5, 5],
  B: [6, 5, 6, 5, 6],
  C: [7, 4, 4, 4, 7],
  D: [6, 5, 5, 5, 6],
  E: [7, 4, 6, 4, 7],
  F: [7, 4, 6, 4, 4],
  G: [7, 4, 5, 5, 7],
  H: [5, 5, 7, 5, 5],
  I: [7, 2, 2, 2, 7],
  J: [1, 1, 1, 5, 7],
  K: [5, 5, 6, 5, 5],
  L: [4, 4, 4, 4, 7],
  M: [5, 7, 7, 5, 5],
  N: [6, 5, 5, 5, 5],
  O: [7, 5, 5, 5, 7],
  P: [6, 5, 6, 4, 4],
  Q: [7, 5, 5, 6, 3],
  R: [6, 5, 6, 5, 5],
  S: [7, 4, 7, 1, 7],
  T: [7, 2, 2, 2, 2],
  U: [5, 5, 5, 5, 7],
  V: [5, 5, 5, 5, 2],
  W: [5, 5, 7, 7, 5],
  X: [5, 5, 2, 5, 5],
  Y: [5, 5, 2, 2, 2],
  Z: [7, 1, 2, 4, 7],
}

function line(img: JimpInstance, x0: number, y0: number, x1: number, y1: number, color: number) {
  let dx = Math.abs(x1 - x0)
  let dy = -Math.abs(y1 - y0)
  const sx = x0 < x1 ? 1 : -1
  const sy = y0 < y1 ? 1 : -1
  let err = dx + dy
  for (;;) {
    if (x0 >= 0 && y0 >= 0 && x0 < img.bitmap.width && y0 < img.bitmap.height) {
      img.setPixelColor(color, x0, y0)
    }
    if (x0 === x1 && y0 === y1) break
    const e2 = 2 * err
    if (e2 >= dy) {
      err += dy
      x0 += sx
    }
    if (e2 <= dx) {
      err += dx
      y0 += sy
    }
  }
}

function circle(img: JimpInstance, cx: number, cy: number, r: number, color: number) {
  for (let y = -r; y <= r; y++) {
    for (let x = -r; x <= r; x++) {
      if (x * x + y * y <= r * r) {
        const px = cx + x
        const py = cy + y
        if (px >= 0 && py >= 0 && px < img.bitmap.width && py < img.bitmap.height) {
          img.setPixelColor(color, px, py)
        }
      }
    }
  }
}

function stampText(img: JimpInstance, text: string, x: number, y: number, color: number) {
  let cx = x
  for (const ch of text.toUpperCase()) {
    const glyph = FONT3X5[ch] ?? [0, 0, 0, 0, 0]
    for (let row = 0; row < 5; row++) {
      const bits = glyph[row]
      for (let col = 0; col < 3; col++) {
        if (bits & (0b100 >> col)) {
          const px = cx + col
          const py = y + row
          if (px >= 0 && py >= 0 && px < img.bitmap.width && py < img.bitmap.height) {
            img.setPixelColor(color, px, py)
          }
        }
      }
    }
    cx += 4
  }
}

async function renderGraphPng(): Promise<Uint8Array> {
  const W = 480
  const H = 320
  const img = new Jimp({ width: W, height: H, color: 0x0d1117ff })

  const nodes = [
    { x: 90, y: 240, c: 0xff79c6ff, label: "wiki" },
    { x: 230, y: 70, c: 0x58a6ffff, label: "index" },
    { x: 250, y: 205, c: 0x3fb950ff, label: "concepts" },
    { x: 380, y: 140, c: 0xe3b341ff, label: "courses" },
    { x: 150, y: 110, c: 0xa371f7ff, label: "raw" },
  ]
  const edges: Array<[number, number]> = [
    [0, 1],
    [0, 2],
    [0, 3],
    [0, 4],
    [1, 2],
    [2, 3],
  ]

  for (const [a, b] of edges) {
    line(img, nodes[a].x, nodes[a].y, nodes[b].x, nodes[b].y, 0x58a6ff88)
  }
  for (const n of nodes) {
    circle(img, n.x, n.y, 13, n.c)
    stampText(img, n.label, n.x - Math.floor(n.label.length * 2), n.y + 20, 0xe6edf3ff)
  }
  stampText(img, "MOCHO DOC GRAPH", 16, 20, 0x58a6ffff)

  const buf = await img.getBuffer("image/png")
  return new Uint8Array(buf)
}

type ViewMode = "browse" | "graph"

function App() {
  const [pages, setPages] = useState<PageRef[]>([])
  const [filtered, setFiltered] = useState<PageRef[]>([])
  const [selected, setSelected] = useState(0)
  const [query, setQuery] = useState("")
  const [page, setPage] = useState<Page | null>(null)
  const [status, setStatus] = useState("loading...")
  const [searchFocused, setSearchFocused] = useState(false)
  const [mode, setMode] = useState<ViewMode>("browse")
  const [graph, setGraph] = useState<Uint8Array | null>(null)
  const [graphError, setGraphError] = useState<string | null>(null)
  const [protocol, setProtocol] = useState<ImageRenderProtocol>("blocks")

  useEffect(() => {
    listPages(BASE_URL)
      .then((ps) => {
        setPages(ps)
        setFiltered(ps)
        setStatus(`${ps.length} pages`)
      })
      .catch((e) => setStatus(`err: ${(e as Error).message}`))
    renderGraphPng()
      .then((png) => {
        setGraph(png)
        setGraphError(null)
      })
      .catch((e) => setGraphError((e as Error).message))
  }, [])

  useEffect(() => {
    const q = query.toLowerCase()
    setSelected(0)
    setFiltered(
      q
        ? pages.filter(
            (p) =>
              p.title.toLowerCase().includes(q) ||
              p.name.toLowerCase().includes(q) ||
              p.summary.toLowerCase().includes(q),
          )
        : pages,
    )
  }, [query, pages])

  useEffect(() => {
    const ref = filtered[selected]
    if (!ref) return
    setStatus(`loading ${ref.name}...`)
    readPage(BASE_URL, ref.name)
      .then((p) => {
        setPage(p)
        setStatus(p.name)
      })
      .catch((e) => setStatus(`err: ${(e as Error).message}`))
  }, [filtered, selected])

  useKeyboard((key) => {
    if (key.name === "q") {
      process.exit(0)
    }
    if (searchFocused) return
    if (key.name === "/") {
      setSearchFocused(true)
    } else if (key.name === "g") {
      setMode((m) => (m === "graph" ? "browse" : "graph"))
    } else if (key.name === "p") {
      setProtocol((p) => (p === "auto" ? "blocks" : p === "blocks" ? "kitty" : p === "kitty" ? "sixel" : "auto"))
    } else if (key.name === "up" || key.name === "k") {
      setSelected((i) => Math.max(0, i - 1))
    } else if (key.name === "down" || key.name === "j") {
      setSelected((i) => Math.min(filtered.length - 1, i + 1))
    }
  })

  return (
    <box flexDirection="column" padding={1} flexGrow={1}>
      <text fg={COLOR_ACCENT} flexShrink={0}>
        mocho — OpenTUI react spike (g: graph, p: image proto, /: search, j/k: move, q: quit)
      </text>
      <box border borderStyle="single" borderColor={COLOR_MUTED} height={3} paddingX={1} flexShrink={0}>
        <input
          focused={searchFocused}
          placeholder="filter pages... (/ to focus, Enter to leave)"
          placeholderColor={COLOR_MUTED}
          textColor={COLOR_FG}
          cursorColor={COLOR_ACCENT}
          onInput={(v) => {
            setQuery(v)
            setSearchFocused(true)
          }}
          onChange={(v) => {
            if (v === "") setQuery("")
          }}
          onSubmit={() => setSearchFocused(false)}
        />
      </box>
      <box flexDirection="row" flexGrow={1} flexShrink={1}>
        <scrollbox
          border
          borderStyle="single"
          borderColor={COLOR_ACCENT}
          title="Pages"
          titleAlignment="left"
          scrollY
          width="33%"
          flexGrow={1}
          flexShrink={1}
        >
          {filtered.map((p, i) => (
            <text key={p.name} fg={i === selected ? COLOR_SELECTED : COLOR_FG} flexShrink={0}>
              {i === selected ? "▸ " : "  "}
              {p.title}
            </text>
          ))}
        </scrollbox>
        <scrollbox
          border
          borderStyle="single"
          borderColor={COLOR_MUTED}
          title={mode === "graph" ? "Graph" : "Page"}
          titleAlignment="left"
          scrollY
          flexGrow={1}
          flexShrink={1}
        >
          {mode === "graph" ? (
            graph ? (
              <image source={graph} fit="fit" protocol={protocol} style={{ width: "100%", height: "100%" }} />
            ) : graphError ? (
              <text fg={COLOR_SELECTED}>graph failed: {graphError}</text>
            ) : (
              <text fg={COLOR_MUTED}>generating graph…</text>
            )
          ) : (
            <markdown content={page?.markdown ?? ""} syntaxStyle={syntaxStyle} width="100%" />
          )}
        </scrollbox>
      </box>
      <text fg={COLOR_MUTED} flexShrink={0}>
        {status}
        {mode === "graph" ? `  [image protocol: ${protocol}]` : ""}
      </text>
    </box>
  )
}

const renderer = await createCliRenderer({ exitOnCtrlC: true, targetFps: 30 })
createRoot(renderer).render(<App />)