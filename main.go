package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
	"github.com/kardianos/service"
	"gopkg.in/yaml.v3"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var logger service.Logger

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	homeDir := getHomeDir()
	configPath := homeDir + "/" + ".gyazo.config.yml"

	LoadConfig(configPath)

	WatchDir()
}
func (p *program) Stop(s service.Service) error {
	return nil
}

type GyazoConfig struct {
	Host          string `yaml:"host"`
	CGI           string `yaml:"cgi"`
	HTTPPort      int    `yaml:"http_port"`
	UseSSL        bool   `yaml:"use_ssl"`
	MarkImportant bool   `yaml:"mark_important"`
	WatchDir      string `yaml:"watch_dir"`
}

var Config GyazoConfig

func LoadConfig(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}
}

func main() {
	svConfig := &service.Config{
		Name:        "GyazoUploader",
		DisplayName: "Gyazo Uploader",
		Description: "Upload image to Gyazo",
	}

	prg := &program{}
	s, err := service.New(prg, svConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "install":
			err := s.Install()
			if err != nil {
				err := logger.Errorf("Install failure: %v", err)
				if err != nil {
					return
				}
			} else {
				err := logger.Info("Install successful.")
				if err != nil {
					return
				}
			}
		case "uninstall":
			err := s.Uninstall()
			if err != nil {
				err := logger.Errorf("Uninstall failure: %v", err)
				if err != nil {
					return
				}
			} else {
				err := logger.Info("Uninstall successful.")
				if err != nil {
					return
				}
			}
		case "start":
			err := s.Start()
			if err != nil {
				err := logger.Errorf("Start failure: %v", err)
				if err != nil {
					return
				}
			} else {
				err := logger.Info("Start successful.")
				if err != nil {
					return
				}
			}
		case "stop":
			err := s.Stop()
			if err != nil {
				err := logger.Errorf("Stop failure: %v", err)
				if err != nil {
					return
				}
			} else {
				err := logger.Info("Stop successful.")
				if err != nil {
					return
				}
			}
		case "restart":
			err := s.Restart()
			if err != nil {
				err := logger.Errorf("Restart failure: %v", err)
				if err != nil {
					return
				}
			} else {
				err := logger.Info("Restart successful.")
				if err != nil {
					return
				}
			}
		default:
			fmt.Println("Command not found. Please use install, uninstall, start, stop, restart")
		}
		return
	}
	
	err = s.Run()
	if err != nil {
		err := logger.Error(err)
		if err != nil {
			return
		}
	}
}

func ShowAlertError(title, text string) {
	log.Println(text)
	if err := beeep.Alert(title, text, ""); err != nil {
		panic(err)
	}
	os.Exit(1)
}

func IsImage(data []byte) bool {
	// Read the first 512 bytes of the data
	buffer := data[:512]

	// Detect the content type (MIME type) of the file
	contentType := http.DetectContentType(buffer)

	// Check if the MIME type indicates an image
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/bmp", "image/tiff":
		return true
	default:
		return false
	}
}

func UploadFile(imageData []byte) error {
	homeDir := getHomeDir()
	activeWindowName := "Gyazo"
	xuri := Config.Host
	boundary := "----BOUNDARYBOUNDARY----"
	ua := "Gyazo/1.2"

	idFile := homeDir + "/" + ".gyazo.id"
	id := ""
	if _, err := os.Stat(idFile); err == nil {
		content, err := os.ReadFile(idFile)
		if err != nil {
			return err
		}
		id = strings.TrimSpace(string(content))
	}

	metadata := map[string]string{
		"app":   activeWindowName,
		"title": activeWindowName,
		"url":   xuri,
		"note":  fmt.Sprintf("%s\n%s", activeWindowName, xuri),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.SetBoundary(boundary)

	w.WriteField("metadata", string(metadataJSON))
	w.WriteField("id", id)
	w.WriteField("important", fmt.Sprintf("%t", Config.MarkImportant))

	fw, err := w.CreateFormFile("imagedata", "gyazo.com")
	if err != nil {
		return err
	}
	_, err = fw.Write(imageData)
	if err != nil {
		return err
	}
	w.Close()

	requestUrl := "https://" + Config.Host + Config.CGI
	req, err := http.NewRequest("POST", requestUrl, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", ua)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	url := string(body)

	err = openURL(url)
	if err != nil {
		return fmt.Errorf("failed to open URL: %v", err)
	}

	newID := resp.Header.Get("X-Gyazo-Id")
	if newID != "" {
		if _, err := os.Stat(filepath.Dir(idFile)); os.IsNotExist(err) {
			err := os.Mkdir(filepath.Dir(idFile), 0755)
			if err != nil {
				return err
			}
		}
	}

	if err := SaveID(newID, idFile); err != nil {
		fmt.Println("Error:", err)
	}
	return nil
}

func SaveID(newID string, idFile string) error {
	if newID != "" {
		dir := filepath.Dir(idFile)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
		}
		if _, err := os.Stat(idFile); err == nil {
			backupName := idFile + time.Now().Format("_20060102150405.bak")
			if err := os.Rename(idFile, backupName); err != nil {
				return fmt.Errorf("failed to rename file: %v", err)
			}
		}
		if err := os.WriteFile(idFile, []byte(newID), 0644); err != nil {
			return fmt.Errorf("failed to write to file: %v", err)
		}
	}
	return nil
}

func WatchDir() {
	watchDir := Config.WatchDir
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Println("New file saved:", event.Name)

					// deplay to wait for file to be written
					time.Sleep(1 * time.Second)

					imageData, err := os.ReadFile(event.Name)
					if err != nil {
						log.Println("Error reading file:", err)
						continue
					}

					if !IsImage(imageData) {
						log.Println("Input data not is image")
						continue
					}

					err = UploadFile(imageData)
					if err != nil {
						log.Println("Error uploading file:", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add(watchDir)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func getHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return usr.HomeDir
}

func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
