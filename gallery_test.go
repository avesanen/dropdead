package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGallery(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	d, err := createTempDropdead()
	require.NoError(err, "NewDropdead should not return error.")
	require.NotNil(d, "New Dropdead should not be nil.")
	defer func() { require.NoError(cleanupTempDropdead(d), "Dropdead should close and cleanup correctly.") }()

	testGallery := &Gallery{
		Name: "dQw4w9WgXcQ",
		Files: []*File{
			&File{Name: "test", Url: "/g/dQw4w9WgXcQ/test"},
		},
	}

	require.NoError(d.SaveGallery(testGallery), "Savegallery should not return an error")

	g, err := d.LoadGallery(testGallery.Name)
	require.NoError(err, "Loadgallery should not return an error.")

	assert.EqualValues(testGallery, g, "Loaded gallery should match saved gallery.")

	g, err = d.LoadGallery("fail")
	require.Error(err, "Loading non existing gallery should return error.")
}
