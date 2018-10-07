package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	os.Remove("test.db")

	fs, err := New("test.db")
	assert.Nil(t, err)

	err = fs.Save(fs.NewPage("test1", "some text"))
	assert.Nil(t, err)
	err = fs.Save(fs.NewPage("test2", "some other thing"))
	assert.Nil(t, err)
	err = fs.DumpSQL()
	assert.Nil(t, err)

	pages, err := fs.Find("some text")
	assert.Nil(t, err)
	fmt.Println(pages)
}
