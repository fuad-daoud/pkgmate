package ui

import (
	"slices"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"

	tea "github.com/charmbracelet/bubbletea"
)

type helpModel struct {
	keys keyMap
	help help.Model
}

type keyMap struct {
	help  key.Binding
	extra []help.KeyMap
}

func (k keyMap) ShortHelp() []key.Binding {
	short := []key.Binding{k.help}
	for _, m := range k.extra {
		short = slices.Concat(short, m.ShortHelp())
	}
	return short
}

func (k keyMap) FullHelp() [][]key.Binding {
	full := [][]key.Binding{{k.help}}
	for _, m := range k.extra {
		full = slices.Concat(full, m.FullHelp())
	}
	return full
}

func NewHelpModel() helpModel {
	var keys = keyMap{
		help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	}
	h := help.New()
	h.FullSeparator = "  "
	return helpModel{
		keys: keys,
		help: h,
	}
}

func (m helpModel) addKeys(keys help.KeyMap) helpModel {
	m.keys.extra = append(m.keys.extra, keys)
	return m
}

func (m helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.help):
			m.help.ShowAll = !m.help.ShowAll
		}
	}
	return m, nil
}

func (m helpModel) View() string {
	return m.help.View(m.keys)
}
