package main

import (
	"embed"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenvan/sfs/utils"

	"github.com/samber/lo"
)

//go:embed templates/*
var templateFS embed.FS

var templates *template.Template

var (
	addr       string
	rootDir    string // 根目录
	workDir    string // 工作目录
	requestURL string // 请求网络地址
	localPath  string // 真实文件路径
	isFolder   bool   // 是否访问目录
)

type FileInfo struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

type PageData struct {
	CurrentPath string
	ParentPath  string
	Files       []FileInfo
}

func init() {
	var err error
	funcMap := template.FuncMap{
		"formatSize": formatSize,
		"formatDate": formatDate,
		"splitPath":  splitPath,
		"joinPath":   joinPath,
	}
	templates, err = template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatal("Failed to parse templates:", err)
	}
}

// formatSize formats file size in human-readable format
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// formatDate formats time in human-readable format
func formatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// splitPath splits a path into components
func splitPath(path string) []string {
	return strings.Split(filepath.Clean(path), string(filepath.Separator))
}

// joinPath joins path components
func joinPath(parts ...string) string {
	return filepath.Join(parts...)
}

func main() {

	// Parse command-line flags
	hostFlag := flag.String("host", "0.0.0.0", "Address to listen on")
	portFlag := flag.String("port", "9527", "Port to listen on")
	dirFlag := flag.String("dir", "", "Working directory to serve files from (default: current directory)")
	flag.Parse()

	// Set address
	addr = fmt.Sprintf("%s:%s", *hostFlag, strings.TrimPrefix(*portFlag, ":"))

	// Set working directory
	var err error
	if *dirFlag != "" {
		rootDir, err = filepath.Abs(*dirFlag)
		if err != nil {
			log.Fatal("Failed to resolve directory path:", err)
		}
		// Check if directory exists
		if info, err := os.Stat(rootDir); err != nil {
			log.Fatal("Directory does not exist:", err)
		} else if !info.IsDir() {
			log.Fatal("Path is not a directory:", rootDir)
		}
	} else {
		rootDir, err = os.Getwd()
		if err != nil {
			log.Fatal("Failed to get working directory:", err)
		}
	}

	http.HandleFunc("/", indexHandler)

	log.Printf("Server starting on http://%s", addr)
	log.Printf("Serving files from: %s", rootDir)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	// Get the request and local path
	requestURL = path.Clean(r.URL.Path)
	localPath = filepath.Clean(filepath.Join(rootDir, requestURL))

	// Security check: ensure the path is within rootDir
	if !strings.HasPrefix(requestURL, "/") {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Invalid path")
		return
	}
	if !strings.HasPrefix(localPath, rootDir) {
		utils.HttpOutput(r, w, http.StatusForbidden, "Access denied")
		return
	}

	isFolder = strings.HasSuffix(requestURL, "/") || utils.IsDir(localPath)
	workDir = map[bool]string{true: localPath, false: filepath.Dir(localPath)}[isFolder]

	// GET:下载(查看文件内容)、POST:上传(保存文件内容)、PUT:调整(移动、复制、改属性)、DELETE:删除
	if !isFolder || r.Method != http.MethodGet {
		functionName := strings.ToLower(r.Method)
		functionMap := map[string]func(w http.ResponseWriter, r *http.Request){
			"get":    getHandler,
			"post":   postHandler,
			"put":    putHandler,
			"delete": deleteHandler,
		}
		if fn, exists := functionMap[functionName]; exists {
			fn(w, r)
			return
		}
		utils.HttpOutput(r, w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// List directory contents
	entries, err := os.ReadDir(localPath)
	if err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error reading directory")
		return
	}

	var files, folders []FileInfo
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}

		item := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(requestURL, entry.Name()),
			Size:    entryInfo.Size(),
			ModTime: entryInfo.ModTime(),
			IsDir:   entry.IsDir(),
		}
		if entry.IsDir() {
			folders = append(folders, item)
		} else {
			files = append(files, item)
		}
	}
	files = append(folders, files...)

	// Calculate parent path
	parentPath := ""
	if requestURL != "" {
		parentPath = filepath.Dir(requestURL)
		if parentPath == "." {
			parentPath = ""
		}
	}

	data := PageData{
		CurrentPath: requestURL,
		ParentPath:  parentPath,
		Files:       files,
	}

	if utils.ReturnJSON(r) {
		fmt.Println(data)
		utils.HttpOutput(r, w, data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Template error: %v", err)
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error rendering page")
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {

	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			utils.HttpOutput(r, w, http.StatusNotFound, "File not found")
			return
		}
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error opening file")
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error getting file info")
		return
	}

	// Don't allow downloading directories
	if fileInfo.IsDir() {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Cannot read directory")
		return
	}

	// Return file content
	if utils.ReturnJSON(r) {
		bytes, err := os.ReadFile(localPath)
		if err != nil {
			utils.HttpOutput(r, w, http.StatusBadRequest, "Cannot read file")
			return
		}
		content := base64.StdEncoding.EncodeToString(bytes)
		utils.HttpOutput(r, w, map[string]interface{}{"content": content})
		return
	}

	// Direct download file
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(localPath)))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		io.Copy(w, file)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {

	if utils.ReturnJSON(r) {
		params, err := utils.HttpInput(r)
		if err != nil {
			utils.HttpOutput(r, w, http.StatusBadRequest, "Error parsing input params: "+err.Error())
			return
		}

		filename, exist := params["filename"]
		if !exist {
			filename = requestURL
		}
		dst := filepath.Clean(filename.(string))
		dstPath := filepath.Join(workDir, dst)
		if strings.HasSuffix(dst, "/") || !strings.HasPrefix(dstPath, rootDir) {
			utils.HttpOutput(r, w, http.StatusForbidden, "Access denied")
			return
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			utils.HttpOutput(r, w, http.StatusInternalServerError, "Error creating directory: "+err.Error())
			return
		}

		content, exist := params["content"]
		if !exist {
			utils.HttpOutput(r, w, "Empty content", http.StatusBadRequest)
			return
		}
		bytes, err := base64.StdEncoding.DecodeString(content.(string))
		if err != nil {
			utils.HttpOutput(r, w, "Error content", http.StatusBadRequest)
			return
		}

		err = os.WriteFile(dstPath, bytes, 0755)
		if err != nil {
			utils.HttpOutput(r, w, http.StatusInternalServerError, "Failed to write file")
			return
		}

		utils.HttpOutput(r, w)
		return
	}

	// Parse multipart form (max 100MB in memory)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Error parsing form: "+err.Error())
		return
	}

	// Get the uploaded file
	srcFile, header, err := r.FormFile("file")
	if err != nil {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Error retrieving file: "+err.Error())
		return
	}
	defer srcFile.Close()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(workDir, 0755); err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error creating directory: "+err.Error())
		return
	}

	// Get filename
	filename := filepath.Base(header.Filename)
	if !isFolder {
		filename = filepath.Base(requestURL)
	}

	// Create destination file
	dstPath := filepath.Join(workDir, filename)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error creating file: "+err.Error())
		return
	}
	defer dstFile.Close()

	// Copy file content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error saving file: "+err.Error())
		return
	}

	utils.HttpOutput(r, w)
}

func putHandler(w http.ResponseWriter, r *http.Request) {

	if !utils.Exist(localPath) {
		utils.HttpOutput(r, w, http.StatusNotFound, "Not found")
		return
	}

	params, err := utils.HttpInput(r)
	if err != nil {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Error parsing input params: "+err.Error())
		return
	}

	act, exist1 := params["act"]
	dst, exist2 := params["dst"]
	if !exist1 || !exist2 {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Lost params")
		return
	}

	action := act.(string)
	if !lo.Contains([]string{"move", "copy"}, action) {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Error action")
		return
	}

	dstName := filepath.Clean(dst.(string))
	dstPath := filepath.Join(map[bool]string{true: rootDir, false: workDir}[strings.HasPrefix(dstName, "/")], dstName)
	if dstName == "/" || !strings.HasPrefix(dstPath, rootDir) {
		utils.HttpOutput(r, w, http.StatusForbidden, "Access denied")
		return
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error creating directory: "+err.Error())
		return
	}
	fmt.Println(dstName, dstPath)

	if action == "move" {
		err = utils.Move(localPath, dstPath)
	} else {
		err = utils.Copy(localPath, dstPath)
	}
	if err != nil {
		utils.HttpOutput(r, w, http.StatusInternalServerError, "Error "+action+": "+err.Error())
		return
	}

	utils.HttpOutput(r, w)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {

	if requestURL == "/" {
		utils.HttpOutput(r, w, http.StatusForbidden, "Forbbidden")
		return
	}

	if !utils.Exist(localPath) {
		utils.HttpOutput(r, w, http.StatusNotFound, "Not found")
		return
	}

	err := os.Remove(localPath)
	if err != nil {
		utils.HttpOutput(r, w, http.StatusBadRequest, "Unable to remove the path:"+requestURL)
		return
	}

	utils.HttpOutput(r, w)
}
