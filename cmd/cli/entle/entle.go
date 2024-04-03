// Package entle uses the [WTFPL] license originally written by [nokusukun].
// A copy is included in the [LICENSE] file
//
// [nokusukun]: https://github.com/nokusukun
// [WTFPL]: https://choosealicense.com/licenses/wtfpl/
// [LICENSE]: https://github.com/ellypaws/inkbunny-app/blob/main/cmd/cli/entle/LICENSE
package entle

import (
	"cmp"
	"sort"
	"strings"
)

type Screen struct {
	Width  int
	Height int
}

type BaseModel struct {
	buffers        map[int]*strings.Builder
	terminal       *Terminal
	topLevelBuffer *strings.Builder
}

func New() BaseModel {
	bm := BaseModel{
		buffers:        make(map[int]*strings.Builder),
		terminal:       NewTerminal(),
		topLevelBuffer: &strings.Builder{},
	}
	return bm
}

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func (m *BaseModel) Render(index int, data string) {
	if _, ok := m.buffers[index]; !ok {
		m.buffers[index] = &strings.Builder{}
	}
	m.buffers[index].WriteString(data)
}

func (m BaseModel) View() string {
	for _, key := range sortedKeys(m.buffers) {
		m.terminal.MoveCursor(0, 0)
		m.terminal.WriteString(m.buffers[key].String())
	}
	m.terminal.Flush()

	return m.terminal.Flush()
}
