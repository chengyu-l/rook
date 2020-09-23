package commons

import (
	"os"
)

// default is true if not set
func IsRookCSIEnableChubaoFS() bool {
	env, ok := os.LookupEnv("ROOK_CSI_ENABLE_CHUBAOFS")
	return !ok || env == "true"
}
