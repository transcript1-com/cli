package version

import "runtime"

var Version = "dev"

func UserAgent() string {
	return "transcript1-cli/" + Version + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
}
