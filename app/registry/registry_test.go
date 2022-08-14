//go:build linux
// +build linux

package registry

import (
	"bytes"
	"testing"
)

func TestCreateDocker(t *testing.T) {
	dockerImage := "alpine:latest"
	docker, err := CreateDocker(dockerImage)
	if err != nil {
		t.Errorf("got an error: %s", err)
	}
	if docker.authToken == "" {
		t.Error("must be authorized")
	}
}

func TestPull(t *testing.T) {
	dockerImage := "alpine:latest"
	docker, _ := CreateDocker(dockerImage)
	if err := docker.Pull(); err != nil {
		t.Errorf("got an error: %s", err)
	}

	dockerImage = "alpine:imagedoesntexist"
	docker, _ = CreateDocker(dockerImage)
	if err := docker.Pull(); err.Error() != "fetch image manifest status code: 404" {
		t.Errorf("image %s mustn't exist", dockerImage)
	}
}

func TestRun(t *testing.T) {
	dockerImage := "alpine:latest"
	command := "echo"
	// -n — to remove a new line
	args := []string{"-n", "hey"}

	docker, _ := CreateDocker(dockerImage)
	if err := docker.Pull(); err != nil {
		t.Errorf("got an error: %s", err)
	}

	fakeStdout := &bytes.Buffer{}
	docker.SetStdout(fakeStdout)
	if err := docker.Run(command, args...); err != nil {
		t.Errorf("got an error: %s", err)
	}

	if fakeStdout.String() != "hey" {
		t.Errorf("got %s, expected %s", fakeStdout.String(), "hey")
	}
}
