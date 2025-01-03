package tui

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	esphome     key.Binding
	quit        key.Binding
	help        key.Binding
	coordinator key.Binding
	graph       key.Binding
	node        key.Binding
	discovery   key.Binding
	firmware    key.Binding
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.esphome, k.quit, k.help, k.coordinator, k.graph, k.node, k.discovery, k.firmware}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.esphome, k.quit, k.help, k.coordinator, k.graph, k.node, k.discovery, k.firmware},
	}
}

func GetKeymap() keymap {
	return keymap{
		node: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("n", "node"),
		),
		firmware: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("f", "firmware"),
		),
		discovery: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("d", "discovery"),
		),
		graph: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("g", "graph"),
		),
		coordinator: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("c", "coordinator"),
		),
		esphome: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("e", "esphome"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
