package macutils

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const KillGrace = 5 * time.Second

func WaitTimeout(c *exec.Cmd, timeout time.Duration) error {
	var TimeoutErr = errors.New("Command timed out.")
	var kill *time.Timer
	term := time.AfterFunc(timeout, func() {
		err := c.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("E! [agent] Error terminating process: %s", err)
			return
		}

		kill = time.AfterFunc(KillGrace, func() {
			err := c.Process.Kill()
			if err != nil {
				log.Printf("E! [agent] Error killing process: %s", err)
				return
			}
		})
	})

	err := c.Wait()

	if kill != nil {
		kill.Stop()
	}
	termSent := !term.Stop()

	// If the process exited without error treat it as success.  This allows a
	// process to do a clean shutdown on signal.
	if err == nil {
		return nil
	}

	// If SIGTERM was sent then treat any process error as a timeout.
	if termSent {
		return TimeoutErr
	}

	// Otherwise there was an error unrelated to termination.
	return err
}

func RunTimeout(c *exec.Cmd, timeout time.Duration) error {
	if err := c.Start(); err != nil {
		return err
	}
	return WaitTimeout(c, timeout)
}

func RunCommandMac(
	command string,
	timeout time.Duration,
	arguments string,
) (string, error) {

	arguments_slice := strings.Fields(arguments)

	// log.Println("D! cmd is ", command, arguments_slice)

	return RunCommand(command, timeout, arguments_slice)
}

func RunCommand(
	command string,
	timeout time.Duration,
	arguments_slice []string,
) (string, error) {

	// log.Println("D! cmd is ", command, arguments_slice)
	cmd := exec.Command(command, arguments_slice...)
	//cmd := exec.Command("system_profiler", "-xml SPHardwareDataType")
	var (
		out bytes.Buffer
		//stderr bytes.Buffer
	)
	cmd.Stdout = &out
	cmd.Stderr = &out

	//log.Println("D! running as command and timeout is", timeout)

	runErr := RunTimeout(cmd, timeout)

	// log.Println("D! output  is >0  ", string(out.Bytes()), runErr)
	return string(out.Bytes()), runErr
}
