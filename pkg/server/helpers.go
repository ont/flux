package server

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func GetenvInt(key string, fallback int) int {
	value, found := os.LookupEnv(key)
	if !found {
		return fallback
	}

	ivalue, err := strconv.Atoi(value)

	if err != nil {
		log.WithError(err).WithField("name", key).Fatal("can't parse env var")
	}

	return ivalue
}

func GetenvStr(key string, fallback string) string {
	value, found := os.LookupEnv(key)
	if !found {
		return fallback
	}

	return value
}
