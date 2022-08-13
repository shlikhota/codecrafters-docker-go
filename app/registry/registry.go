package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Docker struct {
	authToken string
	image     image
}

type image struct {
	repository string
	reference  string
	manifest   imageManifestResponse
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
	imageName := dockerImage
	tag := "latest"
	if strings.Contains(dockerImage, ":") {
		imageName = dockerImage[0:strings.Index(dockerImage, ":")]
		tag = dockerImage[strings.Index(dockerImage, ":")+1:]
	}
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
		return fmt.Errorf("auth status code: %d", response.StatusCode)
	}

	respBody, err := io.ReadAll(response.Body)
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

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch image manifest status code: %d", response.StatusCode)
	}

	manifestResponse := imageManifestResponse{}
	if err := json.Unmarshal(respBody, &manifestResponse); err != nil {
		return err
	}

	d.image.manifest = manifestResponse

	return nil
}

func (d *Docker) Pull(destination string) error {
	for _, layer := range d.image.manifest.FsLayers {
		url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", baseDockerHubUrl, d.image.manifest.Name, layer["blobSum"])
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Docker-Distribution-API-Version", "registry/2.0")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.authToken))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		layerFilename := layer["blobSum"] + ".tar.gz"
		layerFile, err := os.OpenFile(layerFilename, os.O_RDWR|os.O_CREATE, 0750)
		if err != nil {
			return err
		}
		defer layerFile.Close()

		if _, err = io.Copy(layerFile, resp.Body); err != nil {
			return err
		}

		if err := extractTar(layerFile.Name(), destination); err != nil {
			return err
		}
	}

	return nil
}

func extractTar(archiveFilename, destination string) error {
	cmd := exec.Command("tar", []string{"xf", archiveFilename, "-C", destination}...)
	return cmd.Run()
}
