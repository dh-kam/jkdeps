package kotlincompilergolang

import (
	"fmt"
	"path/filepath"
	"time"
)

func resolveRepositoryRoot(root string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root path: %w", err)
	}
	return rootPath, nil
}

func (c *Compiler) collectRepositoryFiles(rootPath string) ([]string, error) {
	return collectKotlinFiles(rootPath, c.config.IncludeKTS, c.config.IncludeBuildScripts)
}

func (c *Compiler) buildRepositoryResult(rootPath string, files []string, startedAt time.Time) RepositoryResult {
	result := newRepositoryResult(rootPath, len(files))
	if len(files) == 0 {
		result.Duration = time.Since(startedAt)
		return result
	}

	for _, unit := range c.collectRepositoryUnits(files) {
		appendRepositoryUnit(&result, unit)
	}
	sortRepositoryUnits(result.Files)
	result.Duration = time.Since(startedAt)
	return result
}
