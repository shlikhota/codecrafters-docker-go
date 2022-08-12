package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	dockerImage := os.Args[2]
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	chrootDir, _ := ioutil.TempDir("", "")

	if docker, err := CreateDocker(dockerImage); docker != nil {
		layers, err := docker.Pull("./")
		if err != nil {
			log.Fatalln(err)
		} else {
			for _, layerFile := range *layers {
				extractTar(layerFile, chrootDir)
			}
		}
	} else {
		log.Fatalln(err)
	}

	if err := createDevNullFile(chrootDir); err != nil {
		log.Fatal(err)
	}

	syscall.Chroot(chrootDir)

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID,
	}

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	} else if err != nil {
		log.Fatal(err)
	}
}

func copyFile(sourceFilepath, destinationPath string) error {
	sourceFile, err := os.Open(sourceFilepath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	stat, _ := sourceFile.Stat()

	destinationFilepath := path.Join(destinationPath, sourceFilepath)
	os.MkdirAll(path.Dir(destinationFilepath), 0750)
	destinationFile, err := os.OpenFile(destinationFilepath, os.O_RDWR|os.O_CREATE, stat.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func createDevNullFile(destinationDir string) error {
	if err := os.MkdirAll(path.Join(destinationDir, "dev"), 0750); err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(destinationDir, "dev", "null"), []byte{}, 0644)
}

func extractTar(archiveFilename, destination string) error {
	cmd := exec.Command("tar", []string{"xf", archiveFilename, "-C", destination}...)
	return cmd.Run()
}
