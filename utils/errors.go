package utils

import (
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/arthurkay/env"
)

func LogError(err error) {
	configDir, err := os.UserConfigDir()
	env.Load(configDir + "/petricoh.conf")
	debug := os.Getenv("DEBUG")
	if debug == "true" {
		if err != nil {
			_, fn, line, _ := runtime.Caller(1)
			log.Printf("[error] %s:%d %v", fn, line, err)
			return
		}
	}

	if err != nil {
		log.Printf("[error] %v", err)
	}
}

func OneLine(in []byte) string {
	str := strings.TrimSpace(string(in))
	return strings.Replace(str, "\n", ". ", -1)
}
