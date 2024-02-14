package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ZIPDownloader struct {
	SourceURL string

	ZipFolderDest   string
	ZipName         string
	AssetFolderDest string

	RemoveAfterDownload bool
}

func (z *ZIPDownloader) DownloadToFile() error {
	// Get the data
	resp, err := http.Get(z.SourceURL)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath.Join(z.ZipFolderDest, z.ZipName))

	if err != nil {
		return err
	}

	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	return err
}

func (z *ZIPDownloader) UnzipToDir() error {
	r, err := zip.OpenReader(filepath.Join(z.ZipFolderDest, z.ZipName))

	if err != nil {
		return err
	}

	defer r.Close()

	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(z.AssetFolderDest, f.Name) // nolint: gosec

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(z.AssetFolderDest)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm) // nolint: errcheck
			continue
		}

		// delete file if exists
		os.Remove(fpath)

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc) // nolint: gosec

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	if z.RemoveAfterDownload {
		os.Remove(filepath.Join(z.ZipFolderDest, z.ZipName))
	}

	return nil
}
