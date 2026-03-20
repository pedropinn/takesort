package trash_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pinn/takesort/internal/trash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteTrash_THMFileDeletedSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	thmFile := filepath.Join(tmpDir, "DJI_0001.THM")
	require.NoError(t, os.WriteFile(thmFile, []byte("thumbnail"), 0o644))

	err := trash.DeleteTrash(thmFile)

	require.NoError(t, err)
	assert.NoFileExists(t, thmFile)
}

func TestDeleteTrash_SRTFileDeletedSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	srtFile := filepath.Join(tmpDir, "DJI_0001.SRT")
	require.NoError(t, os.WriteFile(srtFile, []byte("subtitle data"), 0o644))

	err := trash.DeleteTrash(srtFile)

	require.NoError(t, err)
	assert.NoFileExists(t, srtFile)
}
