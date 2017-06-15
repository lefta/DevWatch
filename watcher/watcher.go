package watcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	fsnotify "gopkg.in/fsnotify.v1"

	action "../action"
)

/*Watcher is the handle to watch API */
type Watcher struct {
	/* Configured vars */
	Actions []action.Action
	Debug   bool
	/* Generated vars */
	dirs        []string
	dirWatcher  *fsnotify.Watcher
	childStatus chan error
}

func (w *Watcher) initFSWatcher() error {
	w.childStatus = make(chan error)

	var err error
	w.dirWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	return filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		// Propagate error
		if err != nil {
			return err
		}

		if f.IsDir() {
			if w.Debug {
				fmt.Println("Debug: Watching", path)
			}
			w.dirs = append(w.dirs, path)
			err = w.dirWatcher.Add(path)
			return err
		}

		return nil
	})
}

/*NewFromJSON creates a watcher from a json definition */
func NewFromJSON(jsonContents []byte) (*Watcher, error) {
	w := &Watcher{}
	err := json.Unmarshal(jsonContents, &w)
	if err != nil {
		return nil, err
	}

	err = w.initFSWatcher()
	return w, err
}

/*Destroy resources that need to */
func (w *Watcher) Destroy() {
	w.dirWatcher.Close()
}

func printDebug(event fsnotify.Event) {
	if event.Op&fsnotify.Create == fsnotify.Create {
		fmt.Println("Debug: Created", event.Name)
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		fmt.Println("Debug: Renamed", event.Name)
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		fmt.Println("Debug: Written", event.Name)
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		fmt.Println("Debug: Removed", event.Name)
	} else {
		fmt.Println("Debug: Unhandled", event)
	}
}

func isDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil {
		return false
	}

	return f.IsDir()
}

func (w *Watcher) handleDirEvent(ev fsnotify.Event) {
	//TODO
	fmt.Println("Unimplemented: Directory event")
}

func (w *Watcher) runActions(ev *fsnotify.Event) {
	for i := range w.Actions {
		if ev == nil || w.Actions[i].Matches(ev.Name) {
			if w.Actions[i].Kill() {
				status := <-w.childStatus
				if w.Debug {
					fmt.Println("Debug:", status)
				}
			}

			err := w.Actions[i].Exec()
			if err != nil {
				fmt.Println(err.Error() + ", waiting for file changes to retry...")
				// Do not execute next actions in case of failure
				break
			}

			go w.Actions[i].Watch(&w.childStatus)
		}
	}
}

/*Run watches */
func (w *Watcher) Run() {
	w.runActions(nil)

	for {
		select {
		case ev := <-w.dirWatcher.Events:
			if w.Debug {
				printDebug(ev)
			}

			if isDir(ev.Name) {
				w.handleDirEvent(ev)
			} else {
				w.runActions(&ev)
			}

		case err := <-w.dirWatcher.Errors:
			fmt.Println("Error:", err)

		case err := <-w.childStatus:
			if err != nil {
				fmt.Println("Program crashed, waiting for file changes...")
			}
		}
	}
}
