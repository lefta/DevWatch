package watcher

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"

	fsnotify "gopkg.in/fsnotify.v1"

	action "../action"
)

/*Watcher is the handle to watch API */
type Watcher struct {
	/* Configured vars */
	Actions []action.Action
	Debug   bool
	//TODO Other hooks
	PostHooks []string
	/* Generated vars */
	dirs        []string
	dirWatcher  *batcher
	childStatus chan error
	postCmds    []*exec.Cmd
}

func (w *Watcher) initFSWatcher() error {
	w.childStatus = make(chan error)
	w.postCmds = make([]*exec.Cmd, len(w.PostHooks))

	var err error
	w.dirWatcher, err = newBatcher()
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
				color.Magenta("Watching", path)
			}
			w.dirs = append(w.dirs, path)
			err = w.dirWatcher.watcher.Add(path)
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

func (w *Watcher) signal(sig syscall.Signal) {
	for _, cmd := range w.postCmds {
		if cmd != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
			// Yay, we have to do it this dirty way, 'cause `go run` makes orphans...
			err := syscall.Kill(cmd.Process.Pid, sig)
			if err != nil {
				color.Red("Failed to kill hook:", err)
			}
		}
	}
	for _, a := range w.Actions {
		a.Kill(sig)
	}
}

/*Destroy resources that need to */
func (w *Watcher) Destroy() {
	w.dirWatcher.close()
	w.signal(syscall.SIGTERM)
}

func printDebug(event fsnotify.Event) {
	if event.Op&fsnotify.Create == fsnotify.Create {
		color.Magenta("Created", event.Name)
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		color.Magenta("Renamed", event.Name)
	} else if event.Op&fsnotify.Write == fsnotify.Write {
		color.Magenta("Written", event.Name)
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		color.Magenta("Removed", event.Name)
	} else {
		color.Magenta("Unhandled", event)
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
	color.Magenta("Unimplemented: Directory event")
}

func (w *Watcher) runActions(evs []fsnotify.Event) {
	// Compute needed actions
	actions := make(map[int]struct{})
	if evs == nil {
		for i := range w.Actions {
			actions[i] = struct{}{}
		}
	} else {
		for _, ev := range evs {
			for i := range w.Actions {
				if w.Actions[i].Matches(ev.Name) {
					actions[i] = struct{}{}
				}
			}
		}
	}

	// Run actions
	for i := range actions {
		if w.Actions[i].Kill(syscall.SIGTERM) {
			status := <-w.childStatus
			if w.Debug {
				color.Magenta(status.Error())
			}
		}

		err := w.Actions[i].Exec()
		if err != nil {
			color.Red(err.Error() + ", waiting for file changes to retry...")
			// Do not execute next actions in case of failure
			break
		}

		go w.Actions[i].Watch(&w.childStatus)
	}
}

func (w *Watcher) runHooks() {
	if w.PostHooks == nil {
		return
	}

	color.Green("Running hooks...")
	for i, cmd := range w.PostHooks {
		fmt.Println(">", cmd)

		argv := strings.Split(cmd, " ")
		w.postCmds[i] = exec.Command(argv[0])
		w.postCmds[i].Args = argv

		w.postCmds[i].Stdout = os.Stdout
		w.postCmds[i].Stderr = os.Stderr

		w.postCmds[i].SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		err := w.postCmds[i].Start()
		if err != nil {
			color.Red("Integration failed")
		}
		//TODO watching
	}
}

/*Run watches */
func (w *Watcher) Run() {
	w.runActions(nil)
	w.runHooks()

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt)

	for {
		select {
		case evs := <-w.dirWatcher.events:
			for _, ev := range evs {
				if w.Debug {
					printDebug(ev)
				}

				if isDir(ev.Name) {
					w.handleDirEvent(ev)
				}
			}

			w.runActions(evs)

		case err := <-w.dirWatcher.watcher.Errors:
			color.Red(err.Error())

		case err := <-w.childStatus:
			if err != nil {
				color.Red("Program crashed, waiting for file changes...")
			}

		case sig := <-sigChannel:
			w.signal(sig.(syscall.Signal))
			w.Destroy()
			os.Exit(0)
		}
	}
}
