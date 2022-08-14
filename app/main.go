package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

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

	docker, err := registry.CreateDocker(dockerImage)
	must(err)
	must(docker.Pull())
	must(docker.Run(command, args...))
}

func must(err error) {
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	} else if err != nil {
		log.Fatal(err)
	}
}
