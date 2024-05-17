package log

import (
	"io"
	"log"
	"os"
	"strconv"
)

var (
	writer   = log.Default().Writer()
	flags    = log.Default().Flags()
	instance = log.New(writer, "JsonapiRouter", flags)
)

func init() {
	debug := os.Getenv("JSONAPI_ROUTER_DEBUG")
	if enabled, err := strconv.ParseBool(debug); err != nil {
		instance.SetOutput(io.Discard)
	} else if !enabled {
		instance.SetOutput(io.Discard)
	}
}

func Printf(format string, args ...any) {
	instance.Printf(format, args...)
}

func Println(args ...any) {
	instance.Println(args...)
}
