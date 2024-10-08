package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

func printError(label *widget.Label, app fyne.App, s string) {
	label.SetText(s)
	time.Sleep(3 * time.Second) //nolint:gomnd,mnd
	app.Quit()
}

func cleanDir(directory string) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil // Skip directories
		}

		// Check if the file has the desired extension
		if strings.HasSuffix(info.Name(), ".exe") || strings.HasSuffix(info.Name(), ".dll") {
			if err = os.Remove(path); err != nil {
				return fmt.Errorf("could not remove file: %s", err.Error())
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not clean directory: %s", err.Error())
	}
	return nil
}

func determineLatestRelease(label *widget.Label) (string, error) {
	label.SetText("Determining latest release")
	httpClient := retryablehttp.NewClient()
	httpClient.Logger = nil
	resp, err := httpClient.Get("https://api.github.com/repos/simple64/simple64/releases/latest")
	if err != nil {
		return "", fmt.Errorf("error determining latest release: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error determining latest release, http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read HTTP response: %s", err.Error())
	}

	var data map[string]interface{}
	if err = json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("error parsing JSON: %s", err.Error())
	}
	simple64Url := ""
	assets, ok := data["assets"].([]interface{})
	if !ok {
		return "", fmt.Errorf("error reading assets")
	}
	for _, element := range assets {
		subArray, ok := element.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("error reading assets")
		}
		name, ok := subArray["name"].(string)
		if !ok {
			return "", fmt.Errorf("error asset file name")
		}
		if strings.Contains(name, "simple64-win64") {
			simple64Url, ok = subArray["browser_download_url"].(string)
			if !ok {
				return "", fmt.Errorf("error reading download url")
			}
		}
	}

	if simple64Url == "" {
		return simple64Url, fmt.Errorf("could not determine download URL")
	}
	return simple64Url, nil
}

func downloadRelease(simple64Url string, label *widget.Label) ([]byte, int64, error) {
	label.SetText("Downloading latest release")
	httpClient := retryablehttp.NewClient()
	httpClient.Logger = nil
	zipResp, err := httpClient.Get(simple64Url)
	if err != nil {
		return nil, 0, fmt.Errorf("error downloading latest release: %s", err.Error())
	}
	defer zipResp.Body.Close()

	if zipResp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("error downloading latest release, http status: %d", zipResp.StatusCode)
	}

	zipBody, err := io.ReadAll(zipResp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("could not read HTTP response: %s", err.Error())
	}
	return zipBody, zipResp.ContentLength, nil
}

func prepDirectory(label *widget.Label) error {
	label.SetText("Cleaning existing directory")

	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(os.Args[1], os.ModePerm); err != nil {
		return fmt.Errorf("could not create directory: %s", err.Error())
	}

	if err := cleanDir(os.Args[1]); err != nil {
		return fmt.Errorf("could not clean existing directory: %s", err.Error())
	}
	return nil
}

func extractZip(label *widget.Label, zipBody []byte, zipLength int64) error {
	label.SetText("Extracting ZIP archive")

	// Open the downloaded zip file
	zipReader, err := zip.NewReader(bytes.NewReader(zipBody), zipLength)
	if err != nil {
		return fmt.Errorf("could not open ZIP: %s", err.Error())
	}

	// Extract each file from the zip archive
	for _, file := range zipReader.File {
		// Open the file from the archive
		zipFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("could not open ZIP file: %s", err.Error())
		}
		defer zipFile.Close()

		if !file.FileInfo().IsDir() {
			// Construct the output file path
			relPath, err := filepath.Rel("simple64", file.Name)
			if err != nil {
				return fmt.Errorf("could not determine file path: %s", err.Error())
			}
			outputPath := filepath.Join(os.Args[1], relPath)

			// Create the parent directory of the file
			if err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
				return fmt.Errorf("could not create directory: %s", err.Error())
			}

			// Create the output file
			outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return fmt.Errorf("could not create file: %s", err.Error())
			}
			defer outputFile.Close()

			// Copy the contents from the zip file to the output file
			for {
				_, err := io.CopyN(outputFile, zipFile, 1024) //nolint:gomnd,mnd
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return fmt.Errorf("could not copy file: %s", err.Error())
				}
			}
		}
	}
	return nil
}

func updateSimple64(label *widget.Label, app fyne.App, c chan bool) {
	time.Sleep(3 * time.Second) //nolint:gomnd,mnd // Wait for simple64-gui to close

	simple64Url, err := determineLatestRelease(label)
	if err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	zipBody, zipLength, err := downloadRelease(simple64Url, label)
	if err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	if err = prepDirectory(label); err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	if err = extractZip(label, zipBody, zipLength); err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	label.SetText("Done extracting ZIP archive")
	time.Sleep(1 * time.Second)
	app.Quit()
	c <- true
}

func main() {
	if len(os.Args) < 2 { //nolint:gomnd,mnd
		log.Fatal("must specify target directory")
	}
	a := app.New()
	w := a.NewWindow("simple64-updater")
	w.Resize(fyne.NewSize(400, 200)) //nolint:gomnd,mnd
	label := widget.NewLabel("Initializing")
	content := container.New(layout.NewCenterLayout(), label)

	w.SetContent(content)

	c := make(chan bool)
	go updateSimple64(label, a, c)
	w.ShowAndRun()

	if <-c {
		cmd := exec.Command(filepath.Join(os.Args[1], "simple64-gui")) //nolint:gosec
		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}
		if err := cmd.Process.Release(); err != nil {
			log.Fatal(err)
		}
	}
}
