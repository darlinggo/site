package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

const projectTmpl = `
+++
date = "{{ .Date }}"
title = "{{ .Name }}"
repo = "{{ .Name }}"
url = "/{{ .Name }}"
+++

{{ .Readme }}
`

var (
	tmpl = template.Must(template.New("project").Parse(projectTmpl))
)

type env struct {
	githubToken string
	hookSecret  []byte
	dir         string
	hugoCmd     string
	hugoConfig  string
	hugoSource  string
	hugoOutput  string
}

func pullReadme(pkg, accessToken string) ([]byte, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/darlinggo/"+pkg+"/readme", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	req.Header.Set("Authorization", "token "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return body, errors.New(pkg + ": non-200 status: " + resp.Status)
	}
	return body, nil
}

type result struct {
	repo string
	body []byte
}

func syncAll(repos []string, token string) map[string][]byte {
	results := map[string][]byte{}
	resultChan := make(chan result)
	var wg sync.WaitGroup
	for _, repo := range repos {
		wg.Add(1)
		go func(r string, wg *sync.WaitGroup, ch chan result) {
			defer wg.Done()
			resp, err := pullReadme(r, token)
			if err != nil {
				log.Println(err)
				return
			}
			ch <- result{body: resp, repo: r}
		}(repo, &wg, resultChan)
	}
	go func(wg *sync.WaitGroup, ch chan result) {
		wg.Wait()
		close(ch)
	}(&wg, resultChan)
	for result := range resultChan {
		results[result.repo] = result.body
	}
	return results
}

func verifyWebhook(mac, body, secret []byte) (bool, error) {
	h := hmac.New(sha1.New, secret)
	_, err := h.Write(body)
	if err != nil {
		return false, err
	}
	expectedMac := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal(mac, []byte(expectedMac)), nil
}

type request struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"repository"`
	Repos []string `json:"repos"`
}

func (e env) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	event := r.Header.Get("X-Github-Event")
	if event != "push" && event != "ping" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ok, err := verifyWebhook([]byte(r.Header.Get("X-Hub-Signature")[5:]), body, e.hookSecret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if event == "ping" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
		return
	}

	var req request
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var readmes map[string][]byte
	if event == "sync-all" {
		readmes = syncAll(req.Repos, e.githubToken)
		w.WriteHeader(http.StatusOK)
		return
	} else {
		ref := strings.Split(req.Ref, "/")
		if len(ref) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		branch := ref[2]
		if branch != "master" {
			w.WriteHeader(http.StatusOK)
			return
		}
		readme, err := pullReadme(req.Repository.Name, e.githubToken)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		readmes = map[string][]byte{req.Repository.Name: readme}
	}

	for repo, readme := range readmes {
		f, err := os.Create(filepath.Join(e.hugoSource, e.dir, repo+".md"))
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		err = tmpl.Execute(f, struct {
			Name   string
			Readme string
			Date   string
		}{
			Name:   repo,
			Readme: string(readme),
			Date:   time.Now().Format(time.RFC3339),
		})
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	output, err := exec.Command(e.hugoCmd, `--config="`+e.hugoConfig+`"`, `--source="`+e.hugoSource+`"`, `--output="`+e.hugoOutput+`"`).CombinedOutput()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println(string(output))
	w.WriteHeader(http.StatusOK)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func main() {
	environment := env{
		dir:         os.ExpandEnv(os.Getenv("OUTPUT_DIR")),
		hookSecret:  []byte(os.Getenv("WEBHOOK_SECRET")),
		githubToken: os.Getenv("GITHUB_TOKEN"),
		hugoCmd:     os.ExpandEnv(os.Getenv("HUGO_CMD")),
		hugoConfig:  os.ExpandEnv(os.Getenv("HUGO_CONFIG")),
		hugoSource:  os.ExpandEnv(os.Getenv("HUGO_SOURCE")),
		hugoOutput:  os.ExpandEnv(os.Getenv("HUGO_OUTPUT")),
	}
	if environment.hookSecret == nil || len(environment.hookSecret) < 1 {
		log.Println("WEBHOOK_SECRET must be set to the secret used to verify webhook requests.")
		os.Exit(1)
	}
	if environment.githubToken == "" {
		log.Println("GITHUB_TOKEN must be set to a personal access token for Github.")
		os.Exit(1)
	}
	if environment.hugoCmd == "" {
		log.Println("HUGO_CMD must be set to the path to the hugo command.")
		os.Exit(1)
	}
	if environment.hugoConfig == "" {
		log.Println("HUGO_CONFIG must be set to the path to the hugo config file to use.")
		os.Exit(1)
	}
	if environment.hugoSource == "" {
		log.Println("HUGO_SOURCE must be set to the root directory of your hugo site.")
		os.Exit(1)
	}
	if environment.hugoOutput == "" {
		log.Println("HUGO_OUTPUT must be set to the directory you'd like the final HTML files written to.")
		os.Exit(1)
	}
	if environment.dir == "" {
		log.Println("OUTPUT_DIR must be set to the directory within " + environment.hugoSource + " to store the project READMEs in.")
		os.Exit(1)
	}
	http.HandleFunc("/health", health)
	http.Handle("/hook", environment)
	err := http.ListenAndServe("0.0.0.0:9001", nil)
	if err != nil {
		panic(err)
	}
}
