package cache

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/kalafut/imohash"
	"github.com/kelseyhightower/envconfig"
)

var hasher imohash.ImoHash
var current File
var settings Settings

type Settings struct {
	RedisURL string `envconfig:"REDIS_URL"`
}

type File struct {
	dir  string
	hash string
	url  string
}

func init() {
	envconfig.Process("", &settings)
	hasher = imohash.New()
}

func Has(cardId, path string) bool {
	current = File{}

	hash, err := hasher.SumFile(path)
	if err != nil {
		log.Warn("couldn't hash file ", path)
		return false
	}
	current.hash = hex.EncodeToString(hash[:])
	current.dir = filepath.Join("/tmp/bt/cache", cardId)
	os.MkdirAll(current.dir, 0777)

	cached, err := ioutil.ReadFile(filepath.Join(current.dir, current.hash))
	if err != nil {
		log.WithFields(log.Fields{
			"file": filepath.Join(current.dir, current.hash),
			"err":  err.Error(),
		}).Warn("couldn't read file from disk")
		return false
	}
	url := string(cached)
	if url != "" {
		current.url = url
		return true
	}
	return false
}

func Url() string {
	return current.url
}

func Save(url string) {
	current.url = url
	err := ioutil.WriteFile(
		filepath.Join(current.dir, current.hash), []byte(current.url), 0777,
	)
	if err != nil {
		log.WithFields(log.Fields{
			"file":  filepath.Join(current.dir, current.hash),
			"value": url,
			"err":   err.Error(),
		}).Warn("couldn't save data on disk")
	}
}
