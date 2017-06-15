package action

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

/*Action stores action details */
type Action struct {
	/* Configured vars */
	Pattern string
	Build   string
	Run     string
	/* Generated vars */
	cmd *exec.Cmd
}

/*Matches the pattern associated to the action with a path */
func (a *Action) Matches(path string) bool {
	_, file := filepath.Split(path)
	matched, err := filepath.Match(a.Pattern, file)

	if err != nil {
		fmt.Println("Error:", err)
		return false
	}

	return matched
}

func newCommand(cmdString string) *exec.Cmd {
	argv := strings.Split(cmdString, " ")
	cmd := exec.Command(argv[0])
	cmd.Args = argv

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

func (a *Action) build() error {
	fmt.Println("Building", a.Pattern)
	fmt.Println(">", a.Build)

	cmd := newCommand(a.Build)

	err := cmd.Run()
	if err != nil {
		return errors.New("Build failed")
	}

	return nil
}

func (a *Action) run() error {
	fmt.Println("Running", a.Run)

	a.cmd = newCommand(a.Run)

	err := a.cmd.Start()
	if err != nil {
		return errors.New("Run failed")
	}

	return nil
}

/*Exec the action, a.k.a. build & run */
func (a *Action) Exec() error {
	if a.Build != "" {
		err := a.build()
		if err != nil {
			return err
		}
	}

	if a.Run != "" {
		err := a.run()
		if err != nil {
			return err
		}
	}
	return nil
}

/*Kill the action */
func (a *Action) Kill() bool {
	if a.cmd != nil && !a.cmd.ProcessState.Exited() {
		// Give the process a chance to exit gracefully
		if a.cmd.Process.Signal(syscall.SIGTERM) != nil {
			// U don't wanna stop? As u want...
			err := a.cmd.Process.Kill()
			if err != nil {
				fmt.Println("Fatal: Failed to kill program:", err)
				os.Exit(1)
			}
		}
		return true
	}

	return false
}

/*Watch the action command for its exit status, sending it to statusChannel. Sends nothing if the
  action is killed and nil on successful exit */
func (a *Action) Watch(statusChannel *chan error) {
	if a.cmd != nil {
		*statusChannel <- a.cmd.Wait()
	}
}
