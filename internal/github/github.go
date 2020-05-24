package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rafaelmartins/github-webhook/internal/archive"
)

func request(url string) (*http.Response, error) {
	token, found := os.LookupEnv("GITHUB_TOKEN")
	if !found {
		return nil, errors.New("GITHUB_TOKEN not defined")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))

	return http.DefaultClient.Do(req)
}

func GetBranchCommitSha(fullName string, branch string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/refs/heads/%s", fullName, branch)
	resp, err := request(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	type ref struct {
		Object struct {
			Type string `json:"type"`
			Sha  string `json:"sha"`
		} `json:"object"`
	}

	r := ref{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	if r.Object.Type == "" {
		return "", fmt.Errorf("Invalid repository (%s) or branch (%s)", fullName, branch)
	}

	if r.Object.Type != "commit" {
		return "", fmt.Errorf("Invalid reference type: %s", r.Object.Type)
	}

	return r.Object.Sha, nil
}

func ExtractCommitFiles(fullName string, sha string, outputDir string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", fullName, sha)
	resp, err := request(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return archive.UnTarGzRootDir(resp.Body, outputDir)
}
