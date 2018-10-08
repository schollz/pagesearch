package dl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadFile(t *testing.T) {
	fileURL := "https://golangcode.com/images/avatar.jpg"

	err := DownloadFile("avatar.jpg", fileURL, 100)
	assert.Nil(t, err)
}
