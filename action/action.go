package action

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
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
		color.Red(err.Error())
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

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}

func (a *Action) build() error {
	color.Green("Building " + a.Pattern)
	fmt.Println(">", a.Build)

	cmd := newCommand(a.Build)

	err := cmd.Run()
	if err != nil {
		return errors.New("Build failed")
	}

	return nil
}

func (a *Action) run() error {
	color.Green("Running " + a.Run)

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
func (a *Action) Kill(sig syscall.Signal) bool {
	if a.cmd != nil && (a.cmd.ProcessState == nil || !a.cmd.ProcessState.Exited()) {
		// Yay, we have to do it this dirty way, 'cause `go run` makes orphans...
		err := syscall.Kill(-a.cmd.Process.Pid, sig)
		if err != nil {
			return false
		}
		a.cmd = nil
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
