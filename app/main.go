package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"

	"docker/registry"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 4 {
		fmt.Println(`Usage:
	your_docker run <image> <command> <arg1> <arg2> ...
		`)
		os.Exit(1)
	}

	dockerImage := os.Args[2]
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	chrootDir, _ := ioutil.TempDir("", "")

	docker, err := registry.CreateDocker(dockerImage)
	must(err)
	must(docker.Pull(chrootDir))

	must(createDevNullFile(chrootDir))
	must(syscall.Chroot(chrootDir))

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID, // linux only
	}

	err = cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	} else if err != nil {
		log.Fatal(err)
	}
}

func must(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func createDevNullFile(destinationDir string) error {
	if err := os.MkdirAll(path.Join(destinationDir, "dev"), 0750); err != nil {
		return err
	}

	return os.WriteFile(path.Join(destinationDir, "dev", "null"), []byte{}, 0644)
}
