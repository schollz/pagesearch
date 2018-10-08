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

	assert.Nil(t,
		fs.SaveMany([]Page{
			{"test3", map[string]string{"url": "hi"}, "something another more thing"},
			{"test4", map[string]string{"url": "hi"}, "and another some thing"},
			{"test5", map[string]string{"url": "hi"}, "one more  another little thing"},
			{"test6", map[string]string{"url": "hi"}, "this is another big thing"},
		}),
	)

	pages, err := fs.Find("some thing")
	assert.Nil(t, err)
	fmt.Println(pages)

	assert.Nil(t, fs.DumpSQL())
}
