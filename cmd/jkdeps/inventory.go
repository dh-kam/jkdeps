package main

import (
	"github.com/dh-kam/jkdeps/internal/flagutil"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func loadExternalIndex(paths []string) ([]string, kcg.ExternalIndex, error) {
	paths = flagutil.UniqueStrings(paths)
	if len(paths) == 0 {
		return paths, kcg.NewExternalIndex(), nil
	}

	index, err := kcg.LoadExternalIndices(paths)
	if err != nil {
		return nil, kcg.ExternalIndex{}, err
	}
	return paths, index, nil
}

func loadExternalIndexFlags(paths stringListFlag) (stringListFlag, kcg.ExternalIndex, error) {
	normalized, index, err := loadExternalIndex(paths)
	if err != nil {
		return nil, kcg.ExternalIndex{}, err
	}
	return stringListFlag(normalized), index, nil
}
