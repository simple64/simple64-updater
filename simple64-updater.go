package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
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
)

func printError(label *widget.Label, app fyne.App, s string) {
	label.SetText(s)
	time.Sleep(3 * time.Second)
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
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func determineLatestRelease(label *widget.Label) (string, error) {
	label.SetText("Determining latest release")

	resp, err := http.Get("https://api.github.com/repos/simple64/simple64/releases/latest")
	if err != nil {
		return "", errors.New("error determining latest release")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("error determining latest release")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("could not read HTTP response")
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", errors.New("error parsing JSON")
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
		return simple64_url, errors.New("could not determine download URL")
	}
	return simple64_url, nil
}

func downloadRelease(simple64_url string, label *widget.Label) ([]byte, int64, error) {
	label.SetText("Downloading latest release")
	zipResp, err := http.Get(simple64_url)
	if err != nil {
		return nil, 0, errors.New("error downloading latest release")
	}
	defer zipResp.Body.Close()

	if zipResp.StatusCode != 200 {
		return nil, 0, errors.New("error downloading latest release")
	}

	zipBody, err := io.ReadAll(zipResp.Body)
	if err != nil {
		return nil, 0, errors.New("could not read HTTP response")
	}
	return zipBody, zipResp.ContentLength, nil
}

func prepDirectory(label *widget.Label) error {
	label.SetText("Cleaning existing directory")

	// Create the output directory if it doesn't exist
	err := os.MkdirAll(os.Args[1], os.ModePerm)
	if err != nil {
		return errors.New("could not create directory")
	}

	err = cleanDir(os.Args[1])
	if err != nil {
		return errors.New("could not clean existing directory")
	}
	return nil
}

func extractZip(label *widget.Label, zipBody []byte, zipLength int64) error {
	label.SetText("Extracting ZIP archive")

	// Open the downloaded zip file
	zipReader, err := zip.NewReader(bytes.NewReader(zipBody), zipLength)
	if err != nil {
		return errors.New("could not open ZIP")
	}

	// Extract each file from the zip archive
	for _, file := range zipReader.File {
		// Open the file from the archive
		zipFile, err := file.Open()
		if err != nil {
			return errors.New("could not open ZIP file")
		}
		defer zipFile.Close()

		// Construct the output file path
		relPath, err := filepath.Rel("simple64", file.Name)
		if err != nil {
			return errors.New("could not determine file path")
		}
		outputPath := filepath.Join(os.Args[1], relPath)

		if !file.FileInfo().IsDir() {
			// Create the parent directory of the file
			err = os.MkdirAll(filepath.Dir(outputPath), os.ModePerm)
			if err != nil {
				return errors.New("could not create directory")
			}

			// Create the output file
			outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return errors.New("could not create file")
			}
			defer outputFile.Close()

			// Copy the contents from the zip file to the output file
			_, err = io.Copy(outputFile, zipFile)
			if err != nil {
				return errors.New("could not write file")
			}
		}
	}
	return nil
}

func updateSimple64(label *widget.Label, app fyne.App, c chan bool) {
	time.Sleep(3 * time.Second) // Wait for simple64-gui to close

	simple64_url, err := determineLatestRelease(label)
	if err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	zipBody, zipLength, err := downloadRelease(simple64_url, label)
	if err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	err = prepDirectory(label)
	if err != nil {
		printError(label, app, err.Error())
		c <- false
		return
	}

	err = extractZip(label, zipBody, zipLength)
	if err != nil {
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
	a := app.New()
	w := a.NewWindow("simple64-updater")
	w.Resize(fyne.NewSize(400, 200))
	label := widget.NewLabel("Initializing")
	content := container.New(layout.NewCenterLayout(), label)

	w.SetContent(content)

	c := make(chan bool)
	go updateSimple64(label, a, c)
	w.ShowAndRun()

	success := <-c
	if success {
		cmd := exec.Command(filepath.Join(os.Args[1], "simple64-gui"))
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		err = cmd.Process.Release()
		if err != nil {
			log.Fatal(err)
		}
	}

}
