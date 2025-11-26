package utils

import (
	"io"
	"os"
	"path/filepath"
)

// 判断文件/目录是否存在
func Exist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// 文件是否存在
func IsFile(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && !info.IsDir()
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && info.IsDir()
}

// 复制文件或文件夹
func Copy(src, dst string) error {
	if IsFile(src) {
		return FileCopy(src, dst)
	}
	if IsDir(src) {
		return DirCopy(src, dst)
	}
	_, err := os.Stat(src)
	return err
}

// 复制文件
func FileCopy(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// 复制目录
func DirCopy(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return FileCopy(path, destPath)
	})
}

// 移动文件
func Move(src, dst string) error {
	err := os.Rename(src, dst)
	return err
}
