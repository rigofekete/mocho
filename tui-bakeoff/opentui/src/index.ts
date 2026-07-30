import {
  createCliRenderer,
  type CliRenderer,
  BoxRenderable,
  TextRenderable,
  InputRenderable,
  InputRenderableEvents,
  ScrollBoxRenderable,
  MarkdownRenderable,
  SyntaxStyle,
  parseColor,
  CliRenderEvents,
  RenderableEvents,
  type ParsedKey,
} from "@opentui/core"
import { listPages, readPage, type PageRef } from "./client.js"

const BASE_URL = process.env.MOCHO_API ?? "http://127.0.0.1:7777"

const COLOR_BG = "#0D1117"
const COLOR_FG = "#E6EDF3"
const COLOR_MUTED = "#6E7681"
const COLOR_ACCENT = "#58A6FF"
const COLOR_SELECTED = "#FF79C6"

const themeStyles = {
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
}

let renderer: CliRenderer | null = null
let root: BoxRenderable | null = null
let header: TextRenderable | null = null
let searchInput: InputRenderable | null = null
let listScroll: ScrollBoxRenderable | null = null
let listBox: BoxRenderable | null = null
let listItems: TextRenderable[] = []
let pageScroll: ScrollBoxRenderable | null = null
let markdown: MarkdownRenderable | null = null
let status: TextRenderable | null = null

let allPages: PageRef[] = []
let filtered: PageRef[] = []
let selected = 0
let searchFocused = false

function applyFilter() {
  const q = (searchInput?.value ?? "").toLowerCase()
  if (q === "") {
    filtered = allPages
    return
  }
  filtered = allPages.filter(
    (p) =>
      p.title.toLowerCase().includes(q) ||
      p.name.toLowerCase().includes(q) ||
      p.summary.toLowerCase().includes(q),
  )
}

function rebuildList() {
  for (const item of listItems) {
    item.destroy()
  }
  listItems = []
  if (!listBox) return

  filtered.forEach((p, i) => {
    const isSel = i === selected
    const item = new TextRenderable(renderer!, {
      id: `item-${i}`,
      content: `${isSel ? "▸ " : "  "}${p.title}`,
      fg: isSel ? parseColor(COLOR_SELECTED) : parseColor(COLOR_FG),
      flexShrink: 0,
    })
    listBox!.add(item)
    listItems.push(item)
  })
}

function restyleList() {
  listItems.forEach((item, i) => {
    const isSel = i === selected
    item.fg = isSel ? parseColor(COLOR_SELECTED) : parseColor(COLOR_FG)
    item.content = `${isSel ? "▸ " : "  "}${filtered[i]?.title ?? ""}`
  })
}

async function loadSelected() {
  if (!markdown || !pageScroll || !status) return
  if (filtered.length === 0) {
    markdown.content = ""
    status.content = "no pages"
    return
  }
  const idx = Math.min(selected, filtered.length - 1)
  const name = filtered[idx].name
  status.content = `loading ${name}...`
  try {
    const page = await readPage(BASE_URL, name)
    markdown.content = page.markdown || ""
    status.content = name
  } catch (e) {
    markdown.content = `**error:** ${(e as Error).message}`
    status.content = `err: ${(e as Error).message}`
  }
}

function onKey(key: ParsedKey) {
  if (key.name === "q" && !key.ctrl) {
    renderer?.destroy()
    return
  }

  if (searchFocused) {
    return
  }

  if (key.name === "/" && !key.ctrl) {
    searchInput?.focus()
    return
  }
  if (key.name === "up" || key.name === "k") {
    if (selected > 0) selected--
    restyleList()
    loadSelected()
    return
  }
  if (key.name === "down" || key.name === "j") {
    if (selected < filtered.length - 1) selected++
    restyleList()
    loadSelected()
    return
  }
}

async function main() {
  renderer = await createCliRenderer({ exitOnCtrlC: true, targetFps: 30 })
  renderer.setBackgroundColor(COLOR_BG)
  renderer.start()

  root = new BoxRenderable(renderer, {
    id: "root",
    padding: 1,
    flexGrow: 1,
    flexDirection: "column",
  })
  renderer.root.add(root)

  header = new TextRenderable(renderer, {
    id: "header",
    content: "mocho — OpenTUI bake-off (/ focus search)",
    fg: parseColor(COLOR_ACCENT),
    flexShrink: 0,
  })
  root.add(header)

  const searchBox = new BoxRenderable(renderer, {
    id: "search-box",
    border: true,
    borderStyle: "single",
    borderColor: parseColor(COLOR_MUTED),
    height: 3,
    flexShrink: 0,
    paddingX: 1,
  })
  root.add(searchBox)

  searchInput = new InputRenderable(renderer, {
    id: "search",
    width: "100%",
    placeholder: "filter pages... (/ to focus)",
    placeholderColor: parseColor(COLOR_MUTED),
    textColor: parseColor(COLOR_FG),
    cursorColor: parseColor(COLOR_ACCENT),
    value: "",
  })
  searchBox.add(searchInput)

  const body = new BoxRenderable(renderer, {
    id: "body",
    flexDirection: "row",
    flexGrow: 1,
    flexShrink: 1,
  })
  root.add(body)

  listScroll = new ScrollBoxRenderable(renderer, {
    id: "list-scroll",
    border: true,
    borderStyle: "single",
    borderColor: parseColor(COLOR_ACCENT),
    title: "Pages",
    titleAlignment: "left",
    scrollY: true,
    width: "33%",
    flexGrow: 1,
    flexShrink: 1,
    paddingX: 0,
    paddingY: 0,
  })
  body.add(listScroll)

  listBox = new BoxRenderable(renderer, {
    id: "list-box",
    flexDirection: "column",
    flexGrow: 1,
    paddingX: 1,
    paddingY: 0,
  })
  listScroll.add(listBox)

  pageScroll = new ScrollBoxRenderable(renderer, {
    id: "page-scroll",
    border: true,
    borderStyle: "single",
    borderColor: parseColor(COLOR_MUTED),
    title: "Page",
    titleAlignment: "left",
    scrollY: true,
    flexGrow: 1,
    flexShrink: 1,
  })
  body.add(pageScroll)

  markdown = new MarkdownRenderable(renderer, {
    id: "markdown",
    content: "",
    syntaxStyle: SyntaxStyle.fromStyles(themeStyles),
    fg: parseColor(COLOR_FG),
    bg: COLOR_BG,
    width: "100%",
  })
  pageScroll.add(markdown)

  status = new TextRenderable(renderer, {
    id: "status",
    content: "loading...",
    fg: parseColor(COLOR_MUTED),
    flexShrink: 0,
  })
  root.add(status)

  searchInput.on(RenderableEvents.FOCUSED, () => {
    searchFocused = true
  })
  searchInput.on(RenderableEvents.BLURRED, () => {
    searchFocused = false
    applyFilter()
    selected = 0
    rebuildList()
    loadSelected()
  })
  searchInput.on(InputRenderableEvents.INPUT, () => {
    applyFilter()
    selected = 0
    rebuildList()
  })

  renderer.keyInput.on("keypress", onKey)

  renderer.on(CliRenderEvents.DESTROY, () => {
    renderer = null
  })

  try {
    allPages = await listPages(BASE_URL)
    filtered = allPages
    status.content = `${allPages.length} pages`
    rebuildList()
    await loadSelected()
  } catch (e) {
    status.content = `err: ${(e as Error).message}`
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})