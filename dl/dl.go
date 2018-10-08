package dl

import (
	"io"
	"net/http"
	"os"
)

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string, maxSize int64) error {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	err = copyMax(out, resp.Body, maxSize)
	if err != nil {
		return err
	}

	return nil
}

func copyMax(dst io.Writer, src io.Reader, n int64) error {
	_, err := io.CopyN(dst, src, n)
	if err != nil {
		// If there's less data available
		// it will throw the error "io.EOF"
		return nil
	}

	// Take one more byte
	// to check if there's more data available
	nextByte := make([]byte, 1)
	nRead, _ := io.ReadFull(src, nextByte)
	if nRead > 0 {
		// Yep, there's too much data
		return nil
	}

	// Exactly the same amount of data
	return nil
}
