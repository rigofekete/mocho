package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/rigofekete/mocho-tui-bakeoff/bubbletea/client"
)

const baseURLDefault = "http://127.0.0.1:7777"

var (
	appStyle    = lipgloss.NewStyle().Padding(0, 1)
	paneBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	cursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("207")).Bold(true)
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	queryStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

type pagesLoadedMsg []client.PageRef
type pageLoadedMsg struct {
	page client.Page
	err  error
}
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type mode int

const (
	modeList mode = iota
	modeSearch
)

type model struct {
	client *client.Client

	pages    []client.PageRef
	filtered []client.PageRef
	index    int

	search textinput.Model
	mode   mode

	viewport     viewport.Model
	renderedText string
	status       string

	width, height int
	ready         bool
}

func initialModel(baseURL string) model {
	ti := textinput.New()
	ti.Placeholder = "filter pages..."
	ti.CharLimit = 64

	vp := viewport.New(60, 20)
	vp.SetContent("")

	return model{
		client:  client.New(baseURL),
		search:  ti,
		viewport: vp,
		status:  "loading...",
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		pages, err := m.client.ListPages(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return pagesLoadedMsg(pages)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		return m, nil
	case pagesLoadedMsg:
		m.pages = []client.PageRef(msg)
		m.filtered = m.pages
		m.status = ""
		if len(m.filtered) == 0 {
			m.status = "no pages in wiki"
		}
		return m, m.loadSelected()
	case pageLoadedMsg:
		if msg.err != nil {
			m.status = "err: " + msg.err.Error()
			m.renderedText = ""
		} else {
			m.status = msg.page.Name
			if r, err := renderMarkdown(msg.page.Markdown, m.contentWidth()); err == nil {
				m.renderedText = r
			} else {
				m.renderedText = msg.page.Markdown
			}
		}
		return m, nil
	case errMsg:
		m.status = "err: " + msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeSearch:
			return m.handleSearchKey(msg)
		default:
			return m.handleListKey(msg)
		}
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m *model) leaveSearch() {
	m.mode = modeList
	m.search.Blur()
	m.applyFilter()
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.leaveSearch()
		return m, nil
	case "enter":
		m.leaveSearch()
		m.index = 0
		return m, m.loadSelected()
	default:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyFilter()
		m.index = 0
		return m, cmd
	}
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "/":
		m.mode = modeSearch
		m.search.Focus()
		return m, textinput.Blink
	case "up", "k":
		if m.index > 0 {
			m.index--
		}
		return m, m.loadSelected()
	case "down", "j":
		if m.index < len(m.filtered)-1 {
			m.index++
		}
		return m, m.loadSelected()
	case "pgup":
		m.index -= 5
		if m.index < 0 {
			m.index = 0
		}
		return m, m.loadSelected()
	case "pgdown":
		m.index += 5
		if m.index > len(m.filtered)-1 {
			m.index = max(len(m.filtered)-1, 0)
		}
		return m, m.loadSelected()
	}
	return m, nil
}

func (m *model) applyFilter() {
	q := strings.ToLower(m.search.Value())
	if q == "" {
		m.filtered = m.pages
		return
	}
	out := make([]client.PageRef, 0, len(m.pages))
	for _, p := range m.pages {
		if strings.Contains(strings.ToLower(p.Title), q) || strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Summary), q) {
			out = append(out, p)
		}
	}
	m.filtered = out
}

func (m *model) loadSelected() tea.Cmd {
	if len(m.filtered) == 0 {
		m.renderedText = ""
		m.status = "no pages"
		return nil
	}
	idx := m.index
	if idx >= len(m.filtered) {
		idx = len(m.filtered) - 1
	}
	name := m.filtered[idx].Name
	c := m.client
	return func() tea.Msg {
		p, err := c.ReadPage(context.Background(), name)
		return pageLoadedMsg{p, err}
	}
}

func (m model) View() string {
	if !m.ready {
		return "starting..."
	}
	listW := m.listWidth()

	list := m.listView()
	pane := paneBorder.Width(listW).Render(list)

	contentW := m.contentWidth()
	m.viewport.Width = contentW
	m.viewport.Height = m.height - 4
	m.viewport.SetContent(m.renderedText)
	viewPane := paneBorder.Width(contentW).Render(titleStyle.Render("Page") + "\n\n" + m.viewport.View())

	bar := m.statusBar()
	body := lipgloss.JoinHorizontal(0, pane, viewPane)
	return appStyle.Render(lipgloss.JoinVertical(0, bar, body, m.helpLine()))
}

func (m model) statusBar() string {
	var s string
	if m.mode == modeSearch {
		s = "search: " + m.search.View()
		return queryStyle.Render(s)
	}
	if m.status != "" {
		return helpStyle.Render(m.status)
	}
	return ""
}

func (m model) helpLine() string {
	return helpStyle.Render("  / search  j/k move  q quit")
}

func (m model) listView() string {
	if len(m.filtered) == 0 {
		return titleStyle.Render("Pages") + "\n\n" + helpStyle.Render("(empty)")
	}
	b := strings.Builder{}
	b.WriteString(titleStyle.Render("Pages"))
	b.WriteString("\n\n")
	for i, p := range m.filtered {
		cursor := "  "
		title := p.Title
		if i == m.index {
			cursor = "▸ "
			b.WriteString(cursorStyle.Render(cursor))
			b.WriteString(selectedStyle.Render(title))
		} else {
			b.WriteString(cursor)
			b.WriteString(title)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) listWidth() int {
	w := m.width / 3
	if w < 24 {
		w = 24
	}
	return w
}

func (m model) contentWidth() int {
	w := m.width - m.listWidth() - 4
	if w < 10 {
		w = 10
	}
	return w
}

func renderMarkdown(md string, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("auto"),
		glamour.WithWordWrap(width-2),
	)
	if err != nil {
		return md, nil
	}
	return r.Render(md)
}

func main() {
	base := baseURLDefault
	if v := os.Getenv("MOCHO_API"); v != "" {
		base = v
	}
	if len(os.Args) > 1 {
		base = os.Args[1]
	}

	p := tea.NewProgram(initialModel(base), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}