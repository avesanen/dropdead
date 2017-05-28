package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func createTempDropdead() (*Dropdead, error) {
	dir, err := ioutil.TempDir("", "dropdead/")
	if err != nil {
		return nil, err
	}

	d, err := NewDropdead(&config{
		Addr:        "127.0.0.1:5000",
		UploadsPath: dir,
		DbPath:      dir,
	})
	return d, nil
}

func cleanupTempDropdead(d *Dropdead) error {
	d.Close()
	if err := os.RemoveAll(d.config.DbPath); err != nil {
		return err
	}
	if err := os.RemoveAll(d.config.UploadsPath); err != nil {
		return err
	}
	return nil
}

func isDir(path string) bool {
	s, err := os.Stat(path)
	if err == nil && s.IsDir() {
		return true
	}
	return false
}

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err == nil && !s.IsDir() {
		return true
	}
	return false
}

func TestNewDropdead(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	d, err := createTempDropdead()
	require.NoError(err, "NewDropdead should not return error.")
	require.NotNil(d, "New Dropdead should not be nil.")
	defer func() { require.NoError(cleanupTempDropdead(d), "Dropdead should close and cleanup correctly.") }()

	assert.True(isDir(d.config.UploadsPath), "Uploads path should exist.")
	assert.True(isDir(d.config.UploadsPath+"/files"), "Uploads path should have a files directory.")
	assert.True(isDir(d.config.DbPath), "Database directory should exist.")
	assert.True(isFile(d.config.DbPath+"/bolt.db"), "Database directory should have bolt.db file.")
}

func TestShutdown(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	d, err := createTempDropdead()
	require.NoError(err, "NewDropdead should not return error.")
	require.NotNil(d, "New Dropdead should not be nil.")
	defer func() { require.NoError(cleanupTempDropdead(d), "Dropdead should close and cleanup correctly.") }()

	errChan := d.ListenAndServe()
	assert.NoError(err, "Listen and serve should not return error.")

	canFail := false
	go func() {
		err := <-errChan
		if canFail {
			assert.IsType(err, io.EOF, "Should return EOF when shutdown gracefully.")
		} else {
			assert.Fail("Dropdead shutdown prematurely.")
		}
	}()

	resp, err := http.Get("http://" + d.config.Addr)
	assert.NoError(err, "GET request to server should not return error.")
	assert.NotNil(resp, "Response from GET request should not be nil.")
	assert.Equal(200, resp.StatusCode, "Server should return 200 OK.")

	canFail = true
	assert.NoError(d.Shutdown(), "Shutdown should return no error.")
}

func TestIndexHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	d, err := createTempDropdead()
	require.NoError(err, "NewDropdead should not return error.")
	require.NotNil(d, "New Dropdead should not be nil.")
	defer func() { require.NoError(cleanupTempDropdead(d), "Dropdead should close and cleanup correctly.") }()

	assert.HTTPSuccess(d.IndexHandler, "GET", "/", nil)
	assert.HTTPBodyContains(d.IndexHandler, "GET", "/", nil, "Drop stuff. Share url.")
}

func TestErrorHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	d, err := createTempDropdead()
	require.NoError(err, "NewDropdead should not return error.")
	require.NotNil(d, "New Dropdead should not be nil.")
	defer func() { require.NoError(cleanupTempDropdead(d), "Dropdead should close and cleanup correctly.") }()

	// Create test http server and client
	ts := httptest.NewServer(d.Mux())
	client := &http.Client{}
	defer ts.Close()

	// Save an empty gallery
	testGallery := &Gallery{
		Name:  "dQw4w9WgXcQ",
		Files: nil,
	}
	assert.NoError(d.SaveGallery(testGallery), "Saving gallery should not return errors")

	failingUrls := []string{
		"/doesnt/exist",
		"/g/nosuchgallery",
		"/g/dQw4w9WgXcQ/nosuchimage.gif",
	}

	for _, url := range failingUrls {
		// Create request
		req, err := http.NewRequest("GET", ts.URL+url, nil)
		assert.NoError(err)

		// Send request
		resp, err := client.Do(req)
		assert.NoError(err, "GET request should complete with error.")
		assert.Equal(418, resp.StatusCode, "Server should be teapot.")
	}

}

func TestUploadHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	d, err := createTempDropdead()
	require.NoError(err, "NewDropdead should not return error.")
	require.NotNil(d, "New Dropdead should not be nil.")
	defer func() { require.NoError(cleanupTempDropdead(d), "Dropdead should close and cleanup correctly.") }()

	// Decode one pixel png for sending.
	img := []byte("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mPkvb/iPwAFKgKVhA22ZgAAAABJRU5ErkJggg==")
	imgBuf := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(img))

	// Set headers
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="files"; filename="pixel.png"`)
	h.Set("Content-Type", "image/png")

	// Add file
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreatePart(h)
	require.NoError(err)

	_, err = io.Copy(fw, imgBuf)
	require.NoError(err)

	w.Close()

	// Create test http server and client
	ts := httptest.NewServer(d.Mux())
	client := &http.Client{}
	defer ts.Close()

	// Create request
	req, err := http.NewRequest("POST", ts.URL+"/upload", &b)
	require.NoError(err)

	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := client.Do(req)
	require.NoError(err, "POST request should complete without errors.")

	// Umarshal and check response
	var respBody uploaderResponse
	dec := json.NewDecoder(resp.Body)
	require.NoError(dec.Decode(&respBody), "Response json should parse.")
	assert.Equal(200, resp.StatusCode, "Response status code should be 200 OK.")
	assert.Equal("ok", respBody.Status, "Response json should have status:\"ok\".")

	// Get uploaded gallery
	galleryUrl := ts.URL + respBody.Url
	req, err = http.NewRequest("GET", galleryUrl, nil)
	require.NoError(err)

	resp, err = client.Do(req)
	require.NoError(err, "GET request should complete without errors.")

	if !assert.Equal(200, resp.StatusCode) {
		assert.FailNow("Status should be 200")
	}

	// Search trough html to find image src.
	var imgSrc string
	z := html.NewTokenizer(resp.Body)
	for tt := z.Next(); imgSrc == ""; tt = z.Next() {
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			assert.Fail("HTML parsing should not return non io.EOF errors.")
			continue
		}
		token := z.Token()
		if token.Data == "img" {
			for _, attr := range token.Attr {
				if attr.Key == "src" {
					imgSrc = attr.Val
					break
				}
			}
		}
	}

	assert.Contains(imgSrc, ".png", "Should find image with png file in src.")

	// Get image
	req, err = http.NewRequest("GET", ts.URL+imgSrc, nil)
	t.Log(ts.URL + imgSrc)
	require.NoError(err)

	resp, err = client.Do(req)
	require.NoError(err, "GET request to image should complete without errors.")
	assert.Equal(200, resp.StatusCode, "Status should be 200.")

	// Read body, encode back to base64 and compare to the original
	fileBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(err)

	data := base64.StdEncoding.EncodeToString(fileBytes)
	assert.Equal(string(img), data, "Uploaded and downloaded image should be equal.")
}
