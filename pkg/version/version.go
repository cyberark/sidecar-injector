package version

import "fmt"

// gitVersion is a SemVer that captures the baked-in version of the sidecar-injector.
const gitVersion = "0.2.0"

// gitCommitShort denotes the specific build type for the sidecar-injector. It defaults to the
// special value 'dev'. It is expected to be replaced by the compile-time value of the
// short sha1 from git of the build commit, output of $(git rev-parse --short HEAD).
var gitCommitShort = "dev"

// Get returns the user-visible aggregation of gitVersion and gitCommitShort
// of this codebase.
func Get() string {
	return fmt.Sprintf("%s-%s", gitVersion, gitCommitShort)
}
