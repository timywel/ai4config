// Package tui 提供 cfg4ai 的终端交互界面（bubbletea）。
// P2 范围：实体浏览 + 导出确认（ARCHITECTURE §12 P2）。
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Item 一个可浏览实体。
type Item struct {
	Kind string
	ID   string
	Note string
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	kindStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type model struct {
	title  string
	items  []Item
	cursor int
}

// NewBrowser 创建实体浏览器。
func NewBrowser(title string, items []Item) *tea.Program {
	return tea.NewProgram(model{title: title, items: items})
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(m.title) + "\n\n")
	for i, it := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}
		sb.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, kindStyle.Render("["+it.Kind+"]"), it.ID, kindStyle.Render(it.Note)))
	}
	sb.WriteString("\n" + kindStyle.Render("↑/k 上 ↓/j 下  q 退出") + "\n")
	return sb.String()
}

// RunBrowser 运行浏览器（阻塞至退出）。
func RunBrowser(title string, items []Item) error {
	p := NewBrowser(title, items)
	_, err := p.Run()
	return err
}
