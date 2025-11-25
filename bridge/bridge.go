package bridge

// build will be overwritten by the linker.
var build = "dev"

// GetVersion returns the version of the binary.
func GetVersion() string {
	return build
}
