package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

type config struct {
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	UploadDir    string `yaml:"upload_dir"`
	AccessPrefix string `yaml:"access_prefix"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

func loalConfig(filepath string) (*config, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("fail to open config file\n%w", err)
	}
	defer file.Close()
	var cfg config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("fail to decode config file\n%w", err)
	}
	return &cfg, nil
}
func basicAuth(r *http.Request, cfg *config) error {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) == 0 {
		return errors.New("authorization header is missing")
	}
	authType, authInfo, ok := strings.Cut(authHeader, " ")
	if !ok || authType != "Basic" {
		return errors.New("invalid authorization type")
	}
	decoded, err := base64.StdEncoding.DecodeString(authInfo)
	if err != nil {
		return fmt.Errorf("failed to decode basic auth info\n%w", err)
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok || username != cfg.Username || password != cfg.Password {
		return errors.New("invalid credentials")
	}
	return nil
}
func uploadHander(w http.ResponseWriter, r *http.Request, cfg *config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := basicAuth(r, cfg)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request: Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	now := time.Now()
	timePath := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day())
	timeNameString := fmt.Sprintf("%s/%s", timePath, filename)
	dirPath := filepath.Join(cfg.UploadDir, timePath)
	filePath := filepath.Join(cfg.UploadDir, timeNameString)
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	url := fmt.Sprintf("%s/%s", cfg.AccessPrefix, timeNameString)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(url))
}
func getHandler(w http.ResponseWriter, r *http.Request, cfg *config) {
	vars := mux.Vars(r)
	year := vars["year"]
	month := vars["month"]
	day := vars["day"]
	filename := vars["filename"]
	ext := filepath.Ext(filename)
	filePath := filepath.Join(cfg.UploadDir, year, month, day, filename)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	contentType := mime.TypeByExtension(ext)
	if len(contentType) == 0 {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=315360000")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))

	http.ServeFile(w, r, filePath)
}
func main() {
	cfg, err := loalConfig("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration\n%v", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadHander(w, r, cfg)
	})
	r.HandleFunc(
		fmt.Sprintf("/%s/{year}/{month}/{day}/{filename}", cfg.AccessPrefix),
		func(w http.ResponseWriter, r *http.Request) {
			getHandler(w, r, cfg)
		},
	)
	hostAndPort := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	log.Printf("the server start listening on %s\n", hostAndPort)
	log.Fatal(http.ListenAndServe(hostAndPort, r))
}
