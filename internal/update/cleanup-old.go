package update

import "os"

const oldSuffix = ".old"

func CleanupOld(exe string) {
	os.Remove(exe + oldSuffix)
}
