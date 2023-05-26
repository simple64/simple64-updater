package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func printError(l *widget.Label, a fyne.App, s string) {
	l.SetText(s)
	time.Sleep(3 * time.Second)
	a.Quit()
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
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func updateSimple64(l *widget.Label, a fyne.App) {
	time.Sleep(3 * time.Second) // Wait for simple64-gui to close
	l.SetText("Determining latest release")

	resp, err := http.Get("https://api.github.com/repos/simple64/simple64/releases/latest")
	if err != nil {
		printError(l, a, "Error determining latest release")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		printError(l, a, "Could not read HTTP response")
		return
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		printError(l, a, "Error parsing JSON")
		return
	}
	simple64_url := ""
	assets := data["assets"].([]interface{})
	for _, element := range assets {
		subArray := element.(map[string]interface{})
		if strings.Contains(subArray["name"].(string), "simple64-win64") {
			simple64_url = subArray["browser_download_url"].(string)
		}
	}

	if simple64_url == "" {
		printError(l, a, "Could not determine download URL")
		return
	}

	l.SetText("Downloading latest release")
	zipResp, err := http.Get(simple64_url)
	if err != nil {
		printError(l, a, "Error downloading latest release")
		return
	}
	defer zipResp.Body.Close()

	// Create the output directory if it doesn't exist
	err = os.MkdirAll(os.Args[1], os.ModePerm)
	if err != nil {
		printError(l, a, "Could not create directory")
		return
	}

	l.SetText("Cleaning existing directory")
	err = cleanDir(os.Args[1])
	if err != nil {
		printError(l, a, "Could not clean existing directory")
		return
	}

	l.SetText("Extracting ZIP archive")
	zipBody, err := io.ReadAll(zipResp.Body)
	if err != nil {
		printError(l, a, "Could not read HTTP response")
		return
	}

	// Open the downloaded zip file
	zipReader, err := zip.NewReader(bytes.NewReader(zipBody), zipResp.ContentLength)
	if err != nil {
		printError(l, a, "Could not open ZIP")
		return
	}

	// Extract each file from the zip archive
	for _, file := range zipReader.File {
		// Open the file from the archive
		zipFile, err := file.Open()
		if err != nil {
			printError(l, a, "Could not open ZIP file")
			return
		}
		defer zipFile.Close()

		// Construct the output file path
		outputPath := filepath.Join(os.Args[1], file.Name)

		if file.FileInfo().IsDir() {
			// Create the directory in the output path
			err = os.MkdirAll(outputPath, os.ModePerm)
			if err != nil {
				printError(l, a, "is this needed?")
				return
			}
		} else {
			// Create the parent directory of the file
			err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
			if err != nil {
				printError(l, a, "Could not create directory")
				return
			}

			// Create the output file
			outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				printError(l, a, "Could not create file")
				return
			}
			defer outputFile.Close()

			// Copy the contents from the zip file to the output file
			_, err = io.Copy(outputFile, zipFile)
			if err != nil {
				printError(l, a, "Could not write file")
				return
			}
		}
	}
	l.SetText("Done extracting ZIP archive")
}

func main() {
	a := app.New()
	w := a.NewWindow("simpl64-updater")
	w.Resize(fyne.NewSize(400, 200))
	label := widget.NewLabel("Initializing")
	content := container.New(layout.NewCenterLayout(), label)

	w.SetContent(content)

	go updateSimple64(label, a)
	w.ShowAndRun()
}
