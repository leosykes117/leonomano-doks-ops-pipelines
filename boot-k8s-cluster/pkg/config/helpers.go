package config

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
)

func getModuleRootPath() (modRootPath string, err error) {
	gomodPath := getGoModPath()
	modRootPath = filepath.Dir(gomodPath)
	if gomodPath == "" {
		err = fmt.Errorf("The go.mod path is empty, cannot determine module root path")
		return
	}
	return
}

func getGoModPath() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return bi.Path
}
