package version

import "fmt"

const (
	Major      = 0
	Minor      = 2
	Patch      = 0
	PreRelease = "alpha"
)

func String() string {
	if PreRelease != "" {
		return fmt.Sprintf("%d.%d.%d-%s", Major, Minor, Patch, PreRelease)
	}
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}

func Short() string {
	return fmt.Sprintf("%d.%d", Major, Minor)
}

func Full() string {
	return fmt.Sprintf("shelp version %s", String())
}
