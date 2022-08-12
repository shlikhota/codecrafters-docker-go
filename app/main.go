package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
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

type image struct {
	repository string
	reference  string
	manifest   imageManifestResponse
}

type Docker struct {
	authToken string
	image     image
}

type authResponse struct {
	Token string `json:"token"`
}

type imageManifestResponse struct {
	Name     string              `json:"name"`
	Tag      string              `json:"tag"`
	FsLayers []map[string]string `json:"fsLayers"`
}

const baseDockerHubUrl = "registry.hub.docker.com"
const dockerAuthUrl string = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"

func CreateDocker(dockerImage string) (*Docker, error) {
	imageName := dockerImage[0:strings.Index(dockerImage, ":")]
	tag := dockerImage[strings.Index(dockerImage, ":")+1:]
	imageName = fmt.Sprintf("library/%s", imageName)
	docker := &Docker{image: image{repository: imageName, reference: tag}}
	if err := docker.auth(); err != nil {
		return nil, err
	}

	if err := docker.fetchImageManifest(); err != nil {
		return nil, err
	}

	return docker, nil
}

func (d *Docker) auth() error {
	url := fmt.Sprintf(dockerAuthUrl, d.image.repository)
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("auth status code: %d\n", response.StatusCode)
	}

	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	authResponse := &authResponse{}
	if err := json.Unmarshal(respBody, authResponse); err != nil {
		return err
	}

	d.authToken = authResponse.Token
	return nil
}

func (d *Docker) fetchImageManifest() error {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", baseDockerHubUrl, d.image.repository, d.image.reference)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.authToken))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch image manifest status code: %d\n", response.StatusCode)
	}

	manifestResponse := imageManifestResponse{}
	if err := json.Unmarshal(respBody, &manifestResponse); err != nil {
		return err
	}

	d.image.manifest = manifestResponse

	return nil
}

func (d *Docker) Pull(destination string) (*[]string, error) {
	var files []string
	for _, layer := range d.image.manifest.FsLayers {
		url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", baseDockerHubUrl, d.image.manifest.Name, layer["blobSum"])
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.authToken))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		layerFilename := path.Join(destination, layer["blobSum"]+".tar.gz")
		layerFile, err := os.OpenFile(layerFilename, os.O_RDWR|os.O_CREATE, 0750)
		if err != nil {
			return nil, err
		}
		defer layerFile.Close()

		_, err = io.Copy(layerFile, resp.Body)
		if err != nil {
			return nil, err
		}

		files = append(files, path.Join(destination, layerFile.Name()))
	}
	return &files, nil
}
