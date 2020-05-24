package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rafaelmartins/github-webhook/internal/exec"
	"github.com/rafaelmartins/github-webhook/internal/github"
)

func run(p *github.Payload) {
	if err := func() error {
		log.Printf("run: %s: Processing payload: %s (%s)", p.Repo.FullName, p.Ref, p.Branch())

		inputDir, err := ioutil.TempDir("", "gw_")
		if err != nil {
			return err
		}
		defer os.RemoveAll(inputDir)

		log.Printf("run: %s: Retrieving files: %s (%s)", p.Repo.FullName, p.Ref, p.After)

		if err := github.ExtractCommitFiles(p.Repo.FullName, p.After, inputDir); err != nil {
			return err
		}

		baseDir, found := os.LookupEnv("GW_BASEDIR")
		if !found {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		if !filepath.IsAbs(baseDir) {
			return errors.New("run: GW_BASEDIR must be absolute path")
		}

		buildId := fmt.Sprintf("%s-%d", p.After, time.Now().Unix())
		outputDir := filepath.Join(baseDir, "builds", buildId)
		if _, err := os.Stat(outputDir); err == nil {
			outputDir += "-"
		}

		symlink := filepath.Join(
			baseDir,
			"htdocs",
			p.Repo.Owner.Login,
			fmt.Sprintf("%s--%s", p.Repo.Name, p.Branch()),
		)

		log.Printf("run: %s: Running website-builder: %s -> %s", p.Repo.FullName, inputDir, outputDir)
		if err := exec.WebsiteBuilder(inputDir, outputDir, symlink); err != nil {
			return err
		}

		log.Printf("run: %s: Success", p.Repo.FullName)

		return nil
	}(); err != nil {
		log.Fatalln("error:", err.Error())
	}
}

func main() {
	l := len(os.Args)

	if l != 2 && l != 3 {
		log.Fatalln("error: invalid number of arguments")
	}

	if l == 3 {

		// calling from command-line
		p, err := github.NewPayloadFromData(os.Args[1], os.Args[2])
		if err != nil {
			log.Fatal(err)
		}

		run(p)

		return
	}

	// web server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p, err := github.NewPayloadFromRequest(r)
		if err != nil {
			log.Println("error: request:", err)
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, fmt.Sprintf("INVALID_PAYLOAD: %s\n", err.Error()))
			return
		}

		if p.Zen != "" {
			log.Printf("request: %s: ping: %s", p.Repo.FullName, p.Zen)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "PONG\n")
			return
		}

		// verify allowed branches only for webhook
		allowed := []string{"master"}
		if a, found := os.LookupEnv("GW_ALLOWED_BRANCHES"); found {
			allowed = strings.Split(a, ",")
		}
		found := false
		for _, b := range allowed {
			if b == "" {
				continue
			}
			if b == p.Branch() {
				found = true
				break
			}
		}
		if !found {
			log.Printf("request: %s: branch not allowed: %s", p.Repo.FullName, p.Branch())
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "BRANCH_NOT_ALLOWED\n")
			return
		}

		// handle deleted branch
		if p.Deleted {
			// TODO
			return
		}

		go run(p)

		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, "ACCEPTED\n")
	})

	if err := http.ListenAndServe(os.Args[1], nil); err != nil {
		log.Fatalln("error:", err)
	}
}
