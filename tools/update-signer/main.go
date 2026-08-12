package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type manifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
	Notes     string `json:"notes,omitempty"`
}

func main() {
	gen := flag.Bool("generate-key", false, "generate an Ed25519 update key pair")
	privatePath := flag.String("private-key", "update-private.key", "base64 private-key file")
	publicPath := flag.String("public-key", "update-public.key", "base64 public-key output")
	version := flag.String("version", "", "release version")
	url := flag.String("url", "", "public HTTPS installer URL")
	file := flag.String("file", "", "installer file to hash")
	notes := flag.String("notes", "", "release notes")
	out := flag.String("out", "update-manifest.json", "manifest output")
	flag.Parse()
	if *gen {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		must(err)
		must(os.WriteFile(*privatePath, []byte(base64.StdEncoding.EncodeToString(priv)+"\n"), 0600))
		must(os.WriteFile(*publicPath, []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0644))
		fmt.Println("Generated update signing keys. Keep the private key offline; configure FormForge with the public key.")
		return
	}
	if *version == "" || *url == "" || *file == "" {
		fmt.Fprintln(os.Stderr, "version, url, and file are required")
		os.Exit(2)
	}
	if !strings.HasPrefix(strings.ToLower(*url), "https://") {
		fmt.Fprintln(os.Stderr, "update URL must use HTTPS")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*file)
	must(err)
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	keyText, err := os.ReadFile(*privatePath)
	must(err)
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyText)))
	must(err)
	if len(key) != ed25519.PrivateKeySize {
		panic("invalid private key")
	}
	msg := []byte(*version + "\n" + *url + "\n" + hash)
	sig := ed25519.Sign(ed25519.PrivateKey(key), msg)
	m := manifest{Version: *version, URL: *url, SHA256: hash, Signature: base64.StdEncoding.EncodeToString(sig), Notes: *notes}
	b, _ := json.MarshalIndent(m, "", "  ")
	must(os.WriteFile(*out, append(b, '\n'), 0644))
	fmt.Println("Wrote", *out)
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
