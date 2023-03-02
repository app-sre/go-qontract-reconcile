package example

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/stretchr/testify/assert"
)

type TestFileInfo struct {
	name    string
	content string
}

func (t *TestFileInfo) Name() string {
	return t.name
}
func (t *TestFileInfo) Size() int64 {
	return int64(len(t.content))
}
func (t *TestFileInfo) Mode() os.FileMode {
	return os.ModePerm
}
func (t *TestFileInfo) ModTime() time.Time {
	return time.Now()
}
func (t *TestFileInfo) IsDir() bool {
	return false
}
func (t *TestFileInfo) Sys() any {
	return nil
}

func TestCurrentEmpty(t *testing.T) {
	e := NewExample()
	called := false

	e.listDirectoryFunc = func(path string) ([]os.FileInfo, error) {
		called = true
		return []os.FileInfo{}, nil
	}

	ri := reconcile.NewResourceInventory()
	ctx := context.Background()

	err := e.CurrentState(ctx, ri)
	assert.NoError(t, err)

	assert.Len(t, ri.State, 0)
	assert.True(t, called)
}

func TestCurrent(t *testing.T) {
	e := NewExample()

	fileInfo := &TestFileInfo{
		name:    "file1",
		content: "content",
	}

	e.listDirectoryFunc = func(path string) ([]os.FileInfo, error) {
		return []os.FileInfo{fileInfo}, nil
	}
	e.readFileFunc = func(path string) ([]byte, error) {
		return []byte(fileInfo.content), nil
	}

	ri := reconcile.NewResourceInventory()
	ctx := context.Background()

	err := e.CurrentState(ctx, ri)

	assert.NoError(t, err)

	state := ri.GetResourceState(fileInfo.Name())

	assert.NotNil(t, state)
	current := state.Current.(*UserFiles)

	assert.Equal(t, fileInfo.Name(), current.FileNames)
	assert.Equal(t, fileInfo.content, current.GpgKey)
}
