package app

import (
	"io"
	"os"
)

func Temp() []byte {
	file, err := os.Open("../app/temp.json")
	if err != nil {
		return nil
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil
	}
	return bytes
}
