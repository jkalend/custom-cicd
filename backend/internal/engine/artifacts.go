package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ArtifactInfo describes a captured artifact file.
type ArtifactInfo struct {
	Name     string `json:"name"`
	StepName string `json:"step_name"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
}

// CollectArtifacts collects files matching patterns for a step and copies them to the run artifact storage directory.
func CollectArtifacts(artifactsDir, runID, stepName string, patterns []string) ([]ArtifactInfo, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	targetDir := filepath.Join(artifactsDir, runID, stepName)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create artifact dir: %w", err)
	}

	var collected []ArtifactInfo
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			fi, err := os.Stat(match)
			if err != nil || fi.IsDir() {
				continue
			}

			destPath := filepath.Join(targetDir, filepath.Base(match))
			if err := copyFile(match, destPath); err != nil {
				continue
			}

			collected = append(collected, ArtifactInfo{
				Name:     filepath.Base(match),
				StepName: stepName,
				Size:     fi.Size(),
				Path:     destPath,
			})
		}
	}
	return collected, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// ListRunArtifacts lists all artifacts stored for a run.
func ListRunArtifacts(artifactsDir, runID string) ([]ArtifactInfo, error) {
	runDir := filepath.Join(artifactsDir, runID)
	var list []ArtifactInfo

	entries, err := os.ReadDir(runDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ArtifactInfo{}, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			stepName := entry.Name()
			stepDir := filepath.Join(runDir, stepName)
			files, err := os.ReadDir(stepDir)
			if err != nil {
				continue
			}
			for _, f := range files {
				if !f.IsDir() {
					info, err := f.Info()
					if err != nil {
						continue
					}
					list = append(list, ArtifactInfo{
						Name:     f.Name(),
						StepName: stepName,
						Size:     info.Size(),
						Path:     filepath.Join(stepDir, f.Name()),
					})
				}
			}
		}
	}
	return list, nil
}
