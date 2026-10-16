package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var DataUrls = []struct {
	Name     string
	Url      string
	FileType string
	Path     string
}{
	{
		Name:     "gtfs",
		Url:      "https://eu.ftp.opendatasoft.com/stif/GTFS/IDFM-gtfs.zip",
		FileType: "zip",
	},
	{
		Name:     "parents_station.csv",
		Url:      "https://data.iledefrance-mobilites.fr/api/explore/v2.1/catalog/datasets/zones-de-correspondance/exports/csv",
		FileType: "csv",
	},
	{
		Name:     "position_in_train.csv",
		Url:      "https://data.iledefrance-mobilites.fr/api/explore/v2.1/catalog/datasets/positionnement-dans-la-rame/exports/csv",
		FileType: "csv",
	},
}

func main() {
	for _, d := range DataUrls {
		if d.Name == "" || d.Url == "" {
			fmt.Println("skip missing data")
			continue
		}
		fmt.Println(d.Name)
		res := fetchData(d.Url)
		var path string = fmt.Sprintf("data/raw/%s", d.Name)
		if d.Path != "" {
			path = d.Path
		}

		switch d.FileType {
		case "zip":
			fmt.Print(saveAndExtractZip(res, "data/raw"))
		default:
			fmt.Print(saveData(res, path))
		}
	}
}

func fetchData(url string) io.Reader {
	resp, err := http.Get(url)
	if err != nil || resp == nil {
		return strings.NewReader("")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		resp.Body.Close()
		return strings.NewReader("")
	}
	return resp.Body
}

func saveData(data io.Reader, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	return err
}

func saveAndExtractZip(data io.Reader, path string) error {
	archive, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	root, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for _, entry := range reader.File {
		destination := filepath.Join(root, filepath.Clean(entry.Name))
		if destination != root && !strings.HasPrefix(destination, root+string(os.PathSeparator)) {
			return fmt.Errorf("invalid ZIP entry path: %s", entry.Name)
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return err
		}

		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.Create(destination)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputErr := input.Close()
		outputErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		if outputErr != nil {
			return outputErr
		}
	}

	return nil
}
