package main

import (
	"fmt"
	"io/ioutil"
	"os"

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
		fmt.Println(err)
		os.Exit(1)
	}

	return json
}

func main() {
	json := getConfig()

	w, err := watcher.NewFromJSON(json)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer w.Destroy()

	w.Run()
}
