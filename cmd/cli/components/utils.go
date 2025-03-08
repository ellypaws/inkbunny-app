package utils

import (
	"bytes"
	"image"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

func IF[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}

func If[T any](condition bool, a, b func() T) T {
	return IF(condition, a, b)()
}

func Propagate[T tea.Model](m T, msg tea.Msg) (T, tea.Cmd) {
	model, cmd := m.Update(msg)
	return model.(T), cmd
}

type RerenderMsg struct{}

var AlwaysRender bool = true

func ForceRender() tea.Cmd {
	if AlwaysRender {
		return func() tea.Msg {
			return RerenderMsg{}
		}
	}
	return nil
}

func AsCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func ToImages(response *entities.TextToImageResponse) ([][]byte, error) {
	return sd.ToImages(response)
}

func ImageSize(b []byte) [2]int {
	if len(b) == 0 {
		return [2]int{-1, -1}
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return [2]int{-1, -1}
	}

	boundSize := img.Bounds().Size()
	return [2]int{boundSize.X, boundSize.Y}
}

func Scale(max, dimensions [2]int) [2]int {
	var maxW = max[0]
	var maxH = max[1]

	originalRatio := float64(dimensions[0]) / float64(dimensions[1])
	maxRatio := float64(maxW) / float64(maxH)

	if originalRatio > maxRatio {
		dimensions[0] = maxW
		dimensions[1] = int(float64(maxW) / originalRatio)
	} else {
		dimensions[1] = maxH
		dimensions[0] = int(float64(maxH) * originalRatio)
	}

	if dimensions[0] > maxW {
		dimensions[0] = maxW
		dimensions[1] = int(float64(maxW) / originalRatio)
	}
	if dimensions[1] > maxH {
		dimensions[1] = maxH
		dimensions[0] = int(float64(maxH) * originalRatio)
	}

	return dimensions
}
