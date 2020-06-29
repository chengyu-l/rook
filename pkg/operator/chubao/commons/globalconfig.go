package commons

import (
	"os"
	"strings"
)

var envMap = make(map[string]string)

func init() {
	// load environments
	environ := os.Environ()
	for _, value := range environ {
		keyValue := strings.Split(value, "=")
		envMap[keyValue[0]] = keyValue[1]
	}
}

const (
	// rook-operator environment keys
	RookCSIEnableChubaoFS = "ROOK_CSI_ENABLE_CHUBAOFS"
)

// default is true if not set
func IsRookCSIEnableChubaoFS() bool {
	enableChubaoFSCSI := envMap[RookCSIEnableChubaoFS]
	return len(enableChubaoFSCSI) == 0 || enableChubaoFSCSI == "true"
}
