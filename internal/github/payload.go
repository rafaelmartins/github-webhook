package github

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Payload struct {
	Zen     string `json:"zen"`
	After   string `json:"after"`
	Deleted bool   `json:"deleted"`
	Ref     string `json:"ref"`
	Repo    struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

func NewPayloadFromData(fullName string, branch string) (*Payload, error) {
	sha, err := GetBranchCommitSha(fullName, branch)
	if err != nil {
		return nil, err
	}

	pieces := strings.Split(fullName, "/")

	pl := &Payload{}
	pl.After = sha
	pl.Deleted = false
	pl.Ref = fmt.Sprintf("refs/heads/%s", branch)
	pl.Repo.Name = pieces[1]
	pl.Repo.FullName = fullName
	pl.Repo.Owner.Login = pieces[0]
	return pl, nil
}

func NewPayloadFromRequest(r *http.Request) (*Payload, error) {
	defer func() {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()

	secret, found := os.LookupEnv("GITHUB_SECRET")
	if !found {
		return nil, errors.New("GITHUB_SECRET not defined")
	}

	if r.Method != http.MethodPost {
		return nil, fmt.Errorf("Invalid HTTP method (%s)", r.Method)
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "" {
		return nil, errors.New("Missing GitHub event")
	}
	if event != "push" && event != "ping" {
		return nil, fmt.Errorf("Invalid event (%s). Only push and ping events supported", event)
	}

	signature := r.Header.Get("X-Hub-Signature")
	if len(signature) == 0 {
		return nil, errors.New("Missing GitHub signature")
	}

	sign := strings.Split(signature, "=")
	if len(sign) != 2 {
		return nil, errors.New("Malformed GitHub signature")
	}

	if sign[0] != "sha1" {
		return nil, fmt.Errorf("Invalid signature algorithm (%s). Only sha1 supported", sign[0])
	}

	mac := hmac.New(sha1.New, []byte(secret))
	tee := io.TeeReader(r.Body, mac)

	var pl Payload
	if err := json.NewDecoder(tee).Decode(&pl); err != nil {
		return nil, err
	}

	sum := mac.Sum(nil)
	actual := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(actual, sum)

	if !hmac.Equal([]byte(sign[1]), actual) {
		return nil, errors.New("Failed to validate HMAC signature")
	}

	// simple data validation, to make sure that child structs were populated
	if pl.Repo.FullName == "" {
		return nil, errors.New("Failed to find repository full name")
	}
	if pl.Repo.Owner.Login == "" {
		return nil, errors.New("Failed to find repository owner login")
	}

	return &pl, nil
}

func (pl *Payload) Branch() string {
	if !strings.HasPrefix(pl.Ref, "refs/heads/") {
		return ""
	}

	return strings.TrimPrefix(pl.Ref, "refs/heads/")
}
