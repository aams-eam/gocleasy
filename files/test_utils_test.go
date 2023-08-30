package files

import (
	"testing"

	"gioui.org/widget"
	"github.com/stretchr/testify/assert"
)

func TestBuildFile(t *testing.T) {
	a := &File{"a", nil, 100, false, []*File{}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	build := NewTestFile("a", 100)
	assert.Equal(t, a, build)
}

func TestBuildFolder(t *testing.T) {
	a := &File{"a", nil, 0, true, []*File{}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	build := NewTestFolder("a")
	assert.Equal(t, a, build)
}

func TestBuildFolderWithFile(t *testing.T) {
	e := &File{"e", nil, 100, false, []*File{}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	d := &File{"d", nil, 100, true, []*File{e}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	e.Parent = d
	build := NewTestFolder("d", NewTestFile("e", 100))
	assert.Equal(t, d, build)
}

func TestBuildComplexFolder(t *testing.T) {
	e := &File{"e", nil, 100, false, []*File{}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	d := &File{"d", nil, 100, true, []*File{e}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	e.Parent = d
	b := &File{"b", nil, 50, false, []*File{}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	c := &File{"c", nil, 100, false, []*File{}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	a := &File{"a", nil, 250, true, []*File{b, c, d}, "", 0, widget.Bool{}, widget.Bool{}, 0}
	b.Parent = a
	c.Parent = a
	d.Parent = a
	build := NewTestFolder("a", NewTestFile("b", 50), NewTestFile("c", 100), NewTestFolder("d", NewTestFile("e", 100)))
	assert.Equal(t, a, build)
}

func TestFindTestFile(t *testing.T) {
	folder := NewTestFolder("a",
		NewTestFolder("b",
			NewTestFile("c", 10),
			NewTestFile("d", 100),
		),
	)
	expected := folder.Files[0].Files[1]
	foundFile := FindTestFile(folder, "d")
	assert.Equal(t, expected, foundFile)
}
