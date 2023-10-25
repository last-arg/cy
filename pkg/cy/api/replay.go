package api

import (
	"fmt"

	"github.com/cfoust/cy/pkg/mux/screen/replay"
	"github.com/cfoust/cy/pkg/mux/screen/tree"
	"github.com/cfoust/cy/pkg/taro"
)

type ReplayModule struct {
}

func (m *ReplayModule) send(context interface{}, msg taro.Msg) error {
	client, ok := context.(Client)
	if !ok {
		return fmt.Errorf("no client could be inferred")
	}

	node := client.Node()
	if node == nil {
		return fmt.Errorf("client was missing node")
	}

	pane, ok := node.(*tree.Pane)
	if !ok {
		return fmt.Errorf("client node was not pane")
	}

	r := pane.ReplayMode()
	if r == nil {
		return fmt.Errorf("client pane was not in replay mode")
	}

	r.Send(msg)
	return nil
}

func (m *ReplayModule) sendAction(context interface{}, action replay.ActionType) error {
	return m.send(context, replay.ActionEvent{
		Type: action,
	})
}

func (m *ReplayModule) Quit(context interface{}) error { return m.send(context, replay.ActionQuit) }

func (m *ReplayModule) ScrollUp(context interface{}) error {
	return m.sendAction(context, replay.ActionScrollUp)
}

func (m *ReplayModule) ScrollDown(context interface{}) error {
	return m.sendAction(context, replay.ActionScrollDown)
}

func (m *ReplayModule) HalfPageUp(context interface{}) error {
	return m.sendAction(context, replay.ActionScrollUpHalf)
}

func (m *ReplayModule) HalfPageDown(context interface{}) error {
	return m.sendAction(context, replay.ActionScrollDownHalf)
}

func (m *ReplayModule) SearchForward(context interface{}) error {
	return m.sendAction(context, replay.ActionSearchForward)
}

func (m *ReplayModule) TimePlay(context interface{}) error {
	return m.sendAction(context, replay.ActionTimePlay)
}

func (m *ReplayModule) SearchAgain(context interface{}) error {
	return m.sendAction(context, replay.ActionSearchAgain)
}

func (m *ReplayModule) SearchReverse(context interface{}) error {
	return m.sendAction(context, replay.ActionSearchReverse)
}

func (m *ReplayModule) SearchBackward(context interface{}) error {
	return m.sendAction(context, replay.ActionSearchBackward)
}

func (m *ReplayModule) TimeStepBack(context interface{}) error {
	return m.sendAction(context, replay.ActionTimeStepBack)
}

func (m *ReplayModule) TimeStepForward(context interface{}) error {
	return m.sendAction(context, replay.ActionTimeStepForward)
}

func (m *ReplayModule) Beginning(context interface{}) error {
	return m.sendAction(context, replay.ActionBeginning)
}

func (m *ReplayModule) End(context interface{}) error {
	return m.sendAction(context, replay.ActionEnd)
}

func (m *ReplayModule) CursorDown(context interface{}) error {
	return m.sendAction(context, replay.ActionCursorDown)
}

func (m *ReplayModule) CursorLeft(context interface{}) error {
	return m.sendAction(context, replay.ActionCursorLeft)
}

func (m *ReplayModule) CursorRight(context interface{}) error {
	return m.sendAction(context, replay.ActionCursorRight)
}

func (m *ReplayModule) CursorUp(context interface{}) error {
	return m.sendAction(context, replay.ActionCursorUp)
}

func (m *ReplayModule) TimePlaybackRate(context interface{}, rate int) error {
	return m.send(context, replay.PlaybackRateEvent{
		Rate: rate,
	})
}