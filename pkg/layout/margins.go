package layout

import (
	"context"

	"github.com/cfoust/cy/pkg/emu"
	"github.com/cfoust/cy/pkg/frames"
	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/geom/tty"
	"github.com/cfoust/cy/pkg/mux"
	S "github.com/cfoust/cy/pkg/mux/screen"
	"github.com/cfoust/cy/pkg/taro"

	"github.com/charmbracelet/lipgloss"
	"github.com/sasha-s/go-deadlock"
)

// Margins puts empty space around a Screen and centers it. In size mode,
// Margins attempts to keep the Screen at a fixed size (one or both dimensions
// fixed). In margin mode, the Screen is surrounded by fixed-size margins that
// do not change with the screen size.
type Margins struct {
	deadlock.RWMutex
	*mux.UpdatePublisher

	screen mux.Screen

	// Whether Margins is in margin mode or size mode
	isMargins bool
	margins   geom.Size
	size      geom.Size

	outer geom.Size
	inner geom.Rect

	borderStyle *lipgloss.Border
}

var _ mux.Screen = (*Margins)(nil)
var _ reusable = (*Margins)(nil)

func fitMargin(outer, margin int) int {
	if margin == 0 {
		return outer
	}

	return geom.Max(outer-(margin*2), 1)
}

func getSize(outer, desired int) int {
	if desired == 0 {
		return outer
	}

	if desired > outer {
		return outer
	}

	return desired
}

func (l *Margins) reuse(node NodeType) (bool, error) {
	config, ok := node.(MarginsType)
	if !ok {
		return false, nil
	}

	l.RLock()
	oldSize := l.size
	l.RUnlock()

	newSize := geom.Vec2{
		R: config.Rows,
		C: config.Cols,
	}

	var changed bool
	if oldSize != newSize {
		l.setSize(newSize)
	}

	if config.Border != nil {
		l.borderStyle = &config.Border.Style
		changed = true
	}

	if !changed {
		return true, nil
	}

	return true, l.recalculate()
}

func (l *Margins) Kill() {
	l.RLock()
	screen := l.screen
	l.RUnlock()

	screen.Kill()
}

func (l *Margins) State() *tty.State {
	l.RLock()
	inner := l.inner
	outer := l.outer
	borderStyle := l.borderStyle
	l.RUnlock()

	innerState := l.screen.State()
	state := tty.New(outer)

	tty.Copy(inner.Position, state, innerState)

	size := state.Image.Size()
	for row := 0; row < size.R; row++ {
		for col := 0; col < size.C; col++ {
			if inner.Contains(geom.Vec2{
				R: row,
				C: col,
			}) {
				continue
			}
			state.Image[row][col].Mode |= emu.AttrTransparent
		}
	}

	if borderStyle != nil {
		left := geom.Clamp(inner.Position.C-1, 0, size.C-1)
		right := geom.Clamp(
			inner.Position.C+inner.Size.C,
			0,
			size.C-1,
		)
		char := []rune(borderStyle.Left)[0]
		for row := 0; row < size.R; row++ {
			state.Image[row][left].Char = char
			state.Image[row][left].Mode ^= emu.AttrTransparent
			state.Image[row][right].Char = char
			state.Image[row][right].Mode ^= emu.AttrTransparent
		}
	}

	return state
}

func (l *Margins) Send(msg mux.Msg) {
	l.RLock()
	inner := l.inner
	l.RUnlock()
	l.screen.Send(taro.TranslateMouseMessage(
		msg,
		-inner.Position.C,
		-inner.Position.R,
	))
}

func (l *Margins) getInner(outer geom.Size) geom.Rect {
	// Resolve the desired margins to a real inner window size
	factor := geom.Size{
		R: fitMargin(outer.R, l.margins.R),
		C: fitMargin(outer.C, l.margins.C),
	}

	// Or just ignore it if there's a predetermined size
	if !l.isMargins {
		factor = l.size
	}

	inner := geom.Size{
		R: geom.Max(getSize(outer.R, factor.R), 1),
		C: geom.Max(getSize(outer.C, factor.C), 1),
	}

	if l.borderStyle != nil {
		inner.C = geom.Max(
			inner.C-2,
			1,
		)
	}

	return geom.Rect{
		Position: outer.Center(inner),
		Size:     inner,
	}
}

func (l *Margins) setSize(size geom.Size) {
	l.isMargins = false
	l.size = geom.Size{
		R: geom.Max(0, size.R),
		C: geom.Max(0, size.C),
	}
}

func (l *Margins) Size() geom.Size {
	l.RLock()
	defer l.RUnlock()
	return l.size
}

func (l *Margins) poll(ctx context.Context) {
	updates := l.screen.Subscribe(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-updates.Recv():
			if _, ok := event.(nodeChangeEvent); ok {
				continue
			}
			l.Publish(event)
		}
	}
}

func (l *Margins) recalculate() error {
	l.RLock()
	outer := l.outer
	l.RUnlock()

	inner := l.getInner(outer)

	l.Lock()
	l.inner = inner
	l.Unlock()

	err := l.screen.Resize(inner.Size)
	if err != nil {
		return err
	}

	return nil
}

func (l *Margins) Resize(size geom.Size) error {
	l.Lock()
	l.outer = size
	l.Unlock()
	return l.recalculate()
}

func (l *LayoutEngine) createMargins(
	node *screenNode,
	config MarginsType,
) error {
	marginsNode, err := l.createNode(
		node.Ctx(),
		config.Node,
	)
	if err != nil {
		return err
	}

	margins := NewMargins(
		node.Ctx(),
		marginsNode.Screen,
	)
	margins.setSize(geom.Size{
		R: config.Rows,
		C: config.Cols,
	})

	if config.Border != nil {
		margins.borderStyle = &config.Border.Style
	}

	node.Screen = margins
	node.Children = []*screenNode{marginsNode}
	return nil
}

func NewMargins(ctx context.Context, screen mux.Screen) *Margins {
	margins := &Margins{
		UpdatePublisher: mux.NewPublisher(),
		size: geom.Size{
			C: 80,
		},
		screen: screen,
	}

	go margins.poll(ctx)

	return margins
}

func AddMargins(ctx context.Context, screen mux.Screen) mux.Screen {
	innerLayers := S.NewLayers()
	innerLayers.NewLayer(
		ctx,
		screen,
		S.PositionTop,
		S.WithOpaque,
		S.WithInteractive,
	)
	margins := NewMargins(ctx, innerLayers)

	outerLayers := S.NewLayers()
	frame := frames.NewFramer(ctx, frames.RandomFrame())
	outerLayers.NewLayer(
		ctx,
		frame,
		S.PositionTop,
	)

	outerLayers.NewLayer(
		ctx,
		margins,
		S.PositionTop,
		S.WithInteractive,
		S.WithOpaque,
	)

	return outerLayers
}