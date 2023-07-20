package fuzzy

import (
	"context"

	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/mux"
	"github.com/cfoust/cy/pkg/mux/screen"
	"github.com/cfoust/cy/pkg/mux/stream"
	"github.com/cfoust/cy/pkg/util"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/xo/terminfo"
)

type Fuzzy struct {
	util.Lifetime
	*screen.Terminal
	info *terminfo.Terminfo
}

var _ mux.Screen = (*Fuzzy)(nil)

func (f *Fuzzy) Resize(size mux.Size) error {
	return nil
}

func NewFuzzy(
	ctx context.Context,
	profile termenv.Profile,
	info *terminfo.Terminfo,
	options []string,
) *Fuzzy {
	renderer := lipgloss.NewRenderer(
		nil,
		termenv.WithProfile(profile),
	)

	lifetime := util.NewLifetime(ctx)

	stream := stream.NewTea(
		ctx,
		initialModel(&lifetime, renderer, options),
		geom.DEFAULT_SIZE,
	)

	terminal := screen.NewTerminal(ctx, stream, geom.DEFAULT_SIZE)

	// TODO(cfoust): 07/20/23 tea interfaces use fake cursor. move this to tea?
	info.Fprintf(terminal, terminfo.CursorInvisible)

	return &Fuzzy{
		Lifetime: lifetime,
		Terminal: terminal,
		info:     info,
	}
}

type model struct {
	lifetime  *util.Lifetime
	options   []string
	matches   []Match
	selected  string
	renderer  *lipgloss.Renderer
	textInput textinput.Model
	queried   string
}

func initialModel(
	lifetime *util.Lifetime,
	renderer *lipgloss.Renderer,
	options []string,
) model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20
	ti.Prompt = ""
	ti.Placeholder = "fuzzy: projects"

	return model{
		lifetime: lifetime,
		options: options,
		// TODO(cfoust): 07/18/23 don't assume
		selected:  options[0],
		renderer:  renderer,
		textInput: ti,
	}
}

var _ tea.Model = (*model)(nil)

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type matchResult struct {
	Matches []Match
}

func queryOptions(options []string, pattern string) tea.Cmd {
	return func() tea.Msg {
		return matchResult{
			Matches: Filter(options, pattern),
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case matchResult:
		m.matches = msg.Matches
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.lifetime.Cancel()
			return m, nil
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	value := m.textInput.Value()
	if m.queried != value {
		m.queried = value
		cmds = append(cmds, queryOptions(m.options, value))
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	basic := m.renderer.NewStyle().
		Background(lipgloss.Color("#20111B")).
		Foreground(lipgloss.Color("#D5CCBA")).
		Width(20)

	inactive := basic.Copy().Background(lipgloss.Color("#968C83"))
	active := basic.Copy().
		Background(lipgloss.Color("#EAA549")).
		Foreground(lipgloss.Color("#20111B"))

	var options []string
	if len(m.matches) != 0 {
		for _, match := range m.matches {
			options = append(options,
				inactive.Render(match.Text),
			)
		}
	} else {
		for _, option := range m.options {
			style := inactive

			if m.selected == option {
				style = active
			}

			options = append(options,
				style.Render(option),
			)
		}
	}

	m.textInput.Cursor.Style = m.renderer.NewStyle().
		Background(lipgloss.Color("#EAA549"))

	return lipgloss.JoinVertical(lipgloss.Left,
		basic.Render(m.textInput.View()),
		lipgloss.JoinVertical(lipgloss.Left, options...),
	)
}