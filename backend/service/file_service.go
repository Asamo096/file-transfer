package service

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileInfo struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
	Path    string
}

type FileService struct {
	ShareDir  string
	OutputDir string
}

func NewFileService(shareDir, outputDir string) *FileService {
	// 确保目录是绝对路径
	absShare, _ := filepath.Abs(shareDir)
	absOutput, _ := filepath.Abs(outputDir)
	return &FileService{
		ShareDir:  absShare,
		OutputDir: absOutput,
	}
}

func (s *FileService) ListFiles(path string) ([]FileInfo, error) {
	if !s.ValidatePath(path) {
		return nil, errPathNotAllowed
	}

	absPath := s.resolvePath(path)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 构建相对路径
		var relPath string
		if path == "/" {
			relPath = "/" + entry.Name()
		} else {
			relPath = path + "/" + entry.Name()
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
			Path:    relPath,
		})
	}

	// 排序：文件夹优先，然后按名称排序
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

func (s *FileService) UploadFile(currentPath string, file *multipart.FileHeader) error {
	// 对于上传，使用输出目录
	// 如果提供了路径，优先使用共享目录
	targetDir := s.ShareDir
	if !s.ValidatePath(currentPath) {
		targetDir = s.OutputDir
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 修复：正确构建目标路径
	var destDir string
	if currentPath == "" || currentPath == "/" {
		destDir = targetDir
	} else {
		safePath := filepath.Clean(currentPath)
		safePath = strings.TrimPrefix(safePath, "/")
		safePath = strings.TrimPrefix(safePath, "\\")
		destDir = filepath.Join(targetDir, safePath)
	}

	// 确保目标目录存在
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// 直接保存文件到目标目录
	destPath := filepath.Join(destDir, file.Filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (s *FileService) DownloadFile(path string) (*os.File, error) {
	if !s.ValidatePath(path) {
		return nil, errPathNotAllowed
	}

	absPath := s.resolvePath(path)
	return os.Open(absPath)
}

func (s *FileService) DeleteFile(path string) error {
	if !s.ValidatePath(path) {
		return errPathNotAllowed
	}

	absPath := s.resolvePath(path)
	return os.RemoveAll(absPath) // 使用 RemoveAll 支持删除非空目录
}

func (s *FileService) RenameFile(oldPath, newPath string) error {
	if !s.ValidatePath(oldPath) || !s.ValidatePath(newPath) {
		return errPathNotAllowed
	}

	absOldPath := s.resolvePath(oldPath)
	absNewPath := s.resolvePath(newPath)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(absNewPath), 0755); err != nil {
		return err
	}

	return os.Rename(absOldPath, absNewPath)
}

func (s *FileService) ReadTextFile(path string) (string, error) {
	if !s.ValidatePath(path) {
		return "", errPathNotAllowed
	}

	absPath := s.resolvePath(path)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *FileService) SaveTextFile(path string, content string) error {
	if !s.ValidatePath(path) {
		return errPathNotAllowed
	}

	absPath := s.resolvePath(path)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

func (s *FileService) CreateDir(path string) error {
	if !s.ValidatePath(path) {
		return errPathNotAllowed
	}

	absPath := s.resolvePath(path)
	return os.MkdirAll(absPath, 0755)
}

func (s *FileService) ValidatePath(path string) bool {
	absPath := s.resolvePath(path)

	absPath = filepath.Clean(absPath)

	// 检查路径是否在允许的目录中
	if strings.HasPrefix(absPath, s.ShareDir) {
		return true
	}
	if strings.HasPrefix(absPath, s.OutputDir) {
		return true
	}

	return false
}

func (s *FileService) resolvePath(path string) string {
	// 如果是空或根路径，直接返回共享目录
	if path == "" || path == "/" {
		return s.ShareDir
	}

	// 清理路径，移除开头的斜杠
	cleanPath := filepath.Clean(path)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	cleanPath = strings.TrimPrefix(cleanPath, "\\")

	// 构建绝对路径
	return filepath.Join(s.ShareDir, cleanPath)
}

var errPathNotAllowed = &pathNotAllowedError{}

type pathNotAllowedError struct{}

func (e *pathNotAllowedError) Error() string {
	return "path not allowed"
}