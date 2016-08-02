package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type request struct {
	Repos []string `json:"repos"`
}

func main() {
	secret := os.Getenv("GITHUB_SECRET")
	endpoint := os.Getenv("HOOK_URL")
	if secret == "" {
		log.Println("GITHUB_SECRET must be set to the hook's secret.")
		os.Exit(1)
	}
	if endpoint == "" {
		log.Println("HOOK_URL must be set to the hook URL to call.")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		log.Println("Usage: syncall {repo} {repo} {repo}")
		os.Exit(1)
	}
	repos := os.Args[1:]
	log.Println("Syncing repos:", repos)
	b, err := json.Marshal(request{Repos: repos})
	if err != nil {
		panic(err)
	}
	log.Println(string(b))
	buf := bytes.NewBuffer(b)
	h := hmac.New(sha1.New, []byte(secret))
	_, err = h.Write(b)
	if err != nil {
		panic(err)
	}
	mac := hex.EncodeToString(h.Sum(nil))
	req, err := http.NewRequest("POST", endpoint, buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Hub-Signature", "sha1="+mac)
	req.Header.Set("X-Github-Event", "sync-all")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(resp.Status+"\n", body)
}
