//go:build enable_e2e_tests

package teste2e

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/FlyInkTV/FlyInkStream-Engine/internal/core"
	"github.com/FlyInkTV/FlyInkStream-Engine/internal/test"
)

func newInstance(conf string) (*core.Core, bool) {
	if conf == "" {
		return core.New([]string{})
	}

	tmpf, err := test.CreateTempFile([]byte(conf))
	if err != nil {
		return nil, false
	}
	defer os.Remove(tmpf)

	return core.New([]string{tmpf})
}

type container struct {
	name string
}

func newContainer(image string, name string, args []string) (*container, error) {
	c := &container{
		name: name,
	}

	exec.Command("docker", "kill", "FlyInkStream-Engine-test-"+name).Run()
	exec.Command("docker", "wait", "FlyInkStream-Engine-test-"+name).Run()

	// --network=host is needed to test multicast
	cmd := []string{
		"docker", "run",
		"--network=host",
		"--name=FlyInkStream-Engine-test-" + name,
		"FlyInkStream-Engine-test-" + image,
	}
	cmd = append(cmd, args...)
	ecmd := exec.Command(cmd[0], cmd[1:]...)
	ecmd.Stdout = nil
	ecmd.Stderr = os.Stderr

	err := ecmd.Start()
	if err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)

	return c, nil
}

func (c *container) close() {
	exec.Command("docker", "kill", "FlyInkStream-Engine-test-"+c.name).Run()
	exec.Command("docker", "wait", "FlyInkStream-Engine-test-"+c.name).Run()
	exec.Command("docker", "rm", "FlyInkStream-Engine-test-"+c.name).Run()
}

func (c *container) wait() int {
	exec.Command("docker", "wait", "FlyInkStream-Engine-test-"+c.name).Run()
	out, _ := exec.Command("docker", "inspect", "FlyInkStream-Engine-test-"+c.name,
		"-f", "{{.State.ExitCode}}").Output()
	code, _ := strconv.ParseInt(string(out[:len(out)-1]), 10, 32)
	return int(code)
}




