// This script is used to generate the dataset files for LLM fine-tuning.
// The fine-tuning goal is to parse non-standard user description into a standard JSON format.
// Text files are used as the input, and the json files are used as the response.
// Output is written to the dataset directory.
//
// Usage:
// go run dataset.go
//
// Before running, make sure to place the text files and json files in the same directory as this script.
// The text files should contain the user descriptions, and the json files should contain the expected responses.
//
// The json files should have the same name as the txt files, but with the .json extension.
//
// Example:
// text file: 1.txt
// json file: 1.json

package main

import (
	"fmt"
	"github.com/ellypaws/inkbunny-sd/utils"
	"log"
	"os"
	"strings"
)

func main() {
	text, json := getFiles()
	dataset := utils.ParseDataset(text, json)

	for name, data := range dataset {
		if _, err := os.Stat("dataset"); os.IsNotExist(err) {
			os.Mkdir("dataset", 0755)
		}
		f, err := os.Create(fmt.Sprintf("dataset/%s.txt", name))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		_, err = f.Write(data)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getFiles() (text, json map[string][]byte) {
	files, _ := os.ReadDir(".")
	for _, f := range files {
		switch {
		case f.IsDir():
			continue
		case strings.HasSuffix(f.Name(), ".txt"):
			b, err := os.ReadFile(f.Name())
			if err != nil {
				continue
			}

			if text == nil {
				text = make(map[string][]byte)
			}

			text[strings.TrimSuffix(f.Name(), ".txt")] = b
		case strings.HasSuffix(f.Name(), ".json"):
			b, err := os.ReadFile(f.Name())
			if err != nil {
				continue
			}

			if json == nil {
				json = make(map[string][]byte)
			}

			json[strings.TrimSuffix(f.Name(), ".json")] = b
		}
	}
	return text, json
}
