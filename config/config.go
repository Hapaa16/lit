package config

// InitDirName is the top-level directory for lit repositories (like .git)
const InitDirName = ".lit"

// DefaultDirs defines the subdirectories created inside InitDirName
var DefaultDirs = []string{
	"objects",
	"refs",
	"refs/heads",
}

// HeadData is the default content of the HEAD file
const HeadData = "ref: refs/heads/main"
