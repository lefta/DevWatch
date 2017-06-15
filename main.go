package main

import (
	"io/ioutil"
	"os"

	"github.com/fatih/color"

	"./watcher"
)

func getConfigFile() string {
	file := ".devwatch.json"

	args := os.Args
	if len(args) >= 2 {
		file = args[1]
	}

	return file
}

func getConfig() []byte {
	file := getConfigFile()
	json, err := ioutil.ReadFile(file)
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}

	return json
}

func main() {
	json := getConfig()

	w, err := watcher.NewFromJSON(json)
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
	defer w.Destroy()

	w.Run()
}
