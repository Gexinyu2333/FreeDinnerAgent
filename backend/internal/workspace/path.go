package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"freedinner/backend/internal/store"
)

func (s *Service) resolve(workspace store.UserWorkspace, requestedPath string) (string, string, error) {
	fullPath, relativePath, err := s.resolveRaw(workspace, requestedPath)
	if err != nil {
		return "", "", err
	}
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err == nil && !isWithinRoot(workspace.RootPath, resolved) {
		return "", "", ErrPathOutsideRoot
	}
	return fullPath, relativePath, nil
}

func (s *Service) resolveForWrite(workspace store.UserWorkspace, requestedPath string) (string, string, error) {
	fullPath, relativePath, err := s.resolveRaw(workspace, requestedPath)
	if err != nil {
		return "", "", err
	}
	parent := filepath.Dir(fullPath)
	if _, err := os.Stat(parent); err == nil {
		resolvedParent, err := filepath.EvalSymlinks(parent)
		if err == nil && !isWithinRoot(workspace.RootPath, resolvedParent) {
			return "", "", ErrPathOutsideRoot
		}
	}
	return fullPath, relativePath, nil
}

func (s *Service) resolveRaw(workspace store.UserWorkspace, requestedPath string) (string, string, error) {
	root := filepath.Clean(workspace.RootPath)
	rawPath := strings.TrimSpace(requestedPath)
	if hasParentSegment(rawPath) {
		return "", "", ErrPathOutsideRoot
	}
	cleaned := filepath.Clean(rawPath)
	if cleaned == "." || cleaned == "/" {
		cleaned = "files"
	} else {
		cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
		cleaned = filepath.Join("files", cleaned)
	}
	fullPath := filepath.Clean(filepath.Join(root, cleaned))
	if !isWithinRoot(root, fullPath) {
		return "", "", ErrPathOutsideRoot
	}
	relativePath, err := filepath.Rel(filepath.Join(root, "files"), fullPath)
	if err != nil {
		return "", "", err
	}
	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "." {
		relativePath = ""
	}
	if strings.HasPrefix(relativePath, "../") || relativePath == ".." {
		return "", "", ErrPathOutsideRoot
	}
	return fullPath, relativePath, nil
}

func isWithinRoot(root, path string) bool {
	originalRoot := filepath.Clean(root)
	cleanRoot := originalRoot
	cleanPath := filepath.Clean(path)
	if resolvedRoot, err := filepath.EvalSymlinks(originalRoot); err == nil {
		cleanRoot = resolvedRoot
		if relativeToOriginal, relErr := filepath.Rel(originalRoot, cleanPath); relErr == nil &&
			relativeToOriginal != ".." && !strings.HasPrefix(relativeToOriginal, "../") {
			cleanPath = filepath.Join(resolvedRoot, relativeToOriginal)
		}
	}
	if resolvedPath, err := filepath.EvalSymlinks(cleanPath); err == nil {
		cleanPath = resolvedPath
	}
	relative, err := filepath.Rel(cleanRoot, cleanPath)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, "../")
}

func hasParentSegment(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
