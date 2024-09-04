package loader

import (
	"context"
	"io"

	"github.com/cfoust/cy/pkg/bind"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/geom/tty"
	"github.com/cfoust/cy/pkg/mux"
	"github.com/cfoust/cy/pkg/replay"
	"github.com/cfoust/cy/pkg/replay/player"
	"github.com/cfoust/cy/pkg/sessions"
	"github.com/cfoust/cy/pkg/taro"
	"github.com/cfoust/cy/pkg/util"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Loader loads a .borg file from disk and opens its contents in a Replay.
type Loader struct {
	util.Lifetime
	path   string
	render *taro.Renderer
	replay *taro.Program
	err    error
	size   geom.Vec2

	timeBinds, copyBinds *bind.BindScope
	options              []replay.Option
}

var _ taro.Model = (*Loader)(nil)

type loadedEvent struct {
	replay *taro.Program
	err    error
}

func (l *Loader) Init() tea.Cmd {
	size := l.size
	return func() tea.Msg {
		reader, err := sessions.Open(l.path)
		if err != nil {
			return loadedEvent{
				err: err,
			}
		}

		events := make([]sessions.Event, 0)
		for {
			event, err := reader.Read()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				return loadedEvent{
					err: err,
				}
			}
			events = append(events, event)
		}

		ctx := l.Lifetime.Ctx()
		replay := replay.New(
			ctx,
			player.FromEvents(events),
			l.timeBinds,
			l.copyBinds,
			l.options...,
		)
		replay.Resize(size)

		return loadedEvent{
			replay: replay,
		}
	}
}

func (l *Loader) View(state *tty.State) {
	if l.replay != nil {
		tty.Copy(geom.Vec2{}, state, l.replay.State())
		return
	}

	var boxContents string
	var borderColor lipgloss.Color

	width := geom.Min(state.Image.Size().C, 50)

	if l.err != nil {
		borderColor = lipgloss.Color("1")
		boxContents = lipgloss.JoinVertical(
			lipgloss.Left,
			l.render.NewStyle().
				Foreground(borderColor).
				Bold(true).
				Render("error"),
			l.render.NewStyle().
				Width(width).
				Render(l.err.Error()),
		)
	} else if l.replay == nil {
		borderColor = lipgloss.Color("4")
		boxContents = lipgloss.JoinVertical(
			lipgloss.Left,
			l.render.NewStyle().
				Foreground(borderColor).
				Bold(true).
				Render("opening .borg file..."),
			l.render.NewStyle().
				Width(width).
				Render(l.path),
		)
	}

	boxStyle := l.render.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	boxText := boxStyle.Render(boxContents)
	l.render.RenderAt(
		state.Image,
		0, 0,
		lipgloss.Place(
			geom.DEFAULT_SIZE.C,
			geom.DEFAULT_SIZE.R,
			lipgloss.Center, lipgloss.Center,
			boxText,
		),
	)
}

func (l *Loader) Update(msg tea.Msg) (taro.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case taro.ScreenUpdate:
		if msg.Msg == nil {
			return l, msg.Wait()
		}

		return l, taro.Sequence(
			func() taro.Msg {
				return taro.PublishMsg{
					Msg: msg.Msg,
				}
			},
			msg.Wait(),
		)
	case tea.WindowSizeMsg:
		l.size = geom.Size{
			R: msg.Height,
			C: msg.Width,
		}

		if l.replay != nil {
			l.replay.Resize(l.size)
		}
		return l, nil
	case replay.ActionEvent, replay.PlaybackRateEvent, taro.KeyMsg, taro.MouseMsg:
		if l.replay == nil {
			return l, nil
		}

		l.replay.Send(msg)
		return l, nil
	case loadedEvent:
		if msg.err != nil {
			l.err = msg.err
			return l, nil
		}

		l.replay = msg.replay
		l.replay.Resize(l.size)

		w := taro.NewWatcher(l.Ctx(), msg.replay)
		return l, w.Wait()
	}

	return l, nil
}

func New(
	ctx context.Context,
	path string,
	timeBinds, copyBinds *bind.BindScope,
	options ...replay.Option,
) mux.Screen {
	l := util.NewLifetime(ctx)
	return taro.New(l.Ctx(), &Loader{
		Lifetime:  l,
		render:    taro.NewRenderer(),
		path:      path,
		options:   options,
		timeBinds: timeBinds,
		copyBinds: copyBinds,
	})
}