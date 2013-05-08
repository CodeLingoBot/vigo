package editor

import (
	"errors"
	"fmt"
	"github.com/kisielk/vigo/buffer"
	"github.com/nsf/termbox-go"
)

type textObjectMode struct {
	editor *editor
	mode   editorMode
	object textObject
	stage  textObjectStage // Text object parsing stage
	err    error           // Set in case of error during text object parsing.

	// Buffer modifier operating on range of cursors; set
	// to run after text object input is complete.
	f buffer.RangeFunc

	outerReps int    // Outer repetitions preceding the initial command.
	repChars  []rune // Temporary buffer for inner repetition digits.
	reps      int    // Inner repetitions
}

type textObjectStage int

// Text object parsing stages
const (
	textObjectStageReps textObjectStage = iota
	textObjectStageChar1
	textObjectStageChar2
)

type textObject struct {
	inner bool
	kind  textObjectKind
}

type textObjectKind int

const (
	textObjectWord textObjectKind = iota
	textObjectWhitespaceWord
	textObjectSentence
	textObjectParagraph
	textObjectSection
	textObjectPercent
	textObjectParens
	textObjectBraces
)

var textObjectKeyToType = map[rune]textObjectKind{
	'w': textObjectWord,
	'W': textObjectWhitespaceWord,
	's': textObjectSentence,
	'p': textObjectParagraph,
	'S': textObjectSection,
	'%': textObjectPercent,
	'b': textObjectParens,
	'B': textObjectBraces,
}

func newTextObjectMode(editor *editor, mode editorMode, f buffer.RangeFunc, reps int) *textObjectMode {
	return &textObjectMode{
		editor:    editor,
		mode:      mode,
		object:    textObject{},
		stage:     textObjectStageReps,
		f:         f,
		outerReps: reps,
	}
}

var ErrBadTextObject error = errors.New("bad text object")

func (m *textObjectMode) onKey(ev *termbox.Event) {
loop:
	switch m.stage {
	case textObjectStageReps:
		if ('0' < ev.Ch && ev.Ch <= '9') || (ev.Ch == '0' && len(m.repChars) > 0) {
			m.repChars = append(m.repChars, ev.Ch)
		} else {
			m.reps = parseReps(string(m.repChars))
			m.stage = textObjectStageChar1
			goto loop
		}
	case textObjectStageChar1:
		switch ev.Ch {
		case 'i':
			m.object.inner = true
		case 'a':
			m.object.inner = false
		default:
			m.stage = textObjectStageChar2
			goto loop
		}
	case textObjectStageChar2:
		if kind, ok := textObjectKeyToType[ev.Ch]; ok {
			m.object.kind = kind
		} else {
			m.err = ErrBadTextObject
		}
		m.editor.setMode(m.mode)
	}
}

func (m *textObjectMode) exit() {
	if m.err != nil {
		// TODO simpler way to get error string?
		m.editor.setStatus(fmt.Sprintf("%v", m.err))
		return
	}

	v := m.editor.active.leaf

	switch m.object.kind {
	case textObjectWord:
		for i := 0; i < m.reps*m.outerReps; i++ {
			from := v.cursor
			to := v.cursor
			// FIXME this wraps onto next line
			if !to.NextWord() {
				v.ctx.setStatus("End of buffer")
			}
			m.f(from, to)
		}
		v.buf.FinalizeActionGroup()
	default:
		m.editor.setStatus("range conversion not implemented")
	}
}
