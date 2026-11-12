package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidFilename(t *testing.T) {
	assert.True(t, ValidFilename("pic.png"))
	assert.False(t, ValidFilename(""))
	assert.False(t, ValidFilename("../secret"))
	assert.False(t, ValidFilename("a/b.png"))
}

func TestMediaFilesStayInsideMediaDir(t *testing.T) {
	t.Chdir(t.TempDir())
	dir, err := GetDir()
	require.NoError(t, err)

	outside := filepath.Join(filepath.Dir(dir), "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "link.txt")))

	_, err = readFile("link.txt")
	require.Error(t, err)

	_, err = OpenFile("../outside.txt")
	require.Error(t, err)

	_, err = CreateFile("../created.txt")
	require.Error(t, err)
	assert.NoFileExists(t, filepath.Join(filepath.Dir(dir), "created.txt"))
}

func TestWriteFileAtomicAndRemove(t *testing.T) {
	t.Chdir(t.TempDir())
	dir, err := GetDir()
	require.NoError(t, err)

	require.NoError(t, writeFileAtomic("out.bin", []byte("data")))
	got, err := readFile("out.bin")
	require.NoError(t, err)
	assert.Equal(t, []byte("data"), got)
	assert.NoFileExists(t, filepath.Join(dir, "out.bin.tmp"))

	info, err := os.Stat(filepath.Join(dir, "out.bin"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	require.NoError(t, RemoveFile("out.bin"))
	assert.NoFileExists(t, filepath.Join(dir, "out.bin"))
}
