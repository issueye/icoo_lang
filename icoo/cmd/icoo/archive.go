package main

import "strings"

const (
	bundleFileExt  = ".icb"
	packageFileExt = ".icpkg"
)

func isArchivePath(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, bundleFileExt) || strings.HasSuffix(lower, packageFileExt)
}
