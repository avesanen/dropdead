package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testConfigString = "addr: 127.0.0.1:1234\ndb_path: /var/dropdead\nuploads_path: /var/dropdead\n"

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "dropdead/")
	assert.NoError(err)

	confFileName := dir + "/config.yml"

	err = ioutil.WriteFile(confFileName, []byte(testConfigString), 0600)
	assert.NoError(err)

	conf, err := loadConfig(confFileName)
	assert.NoError(err, "loadConfig should load without error.")
	assert.Equal("127.0.0.1:1234", conf.Addr, "Addr should load correctly.")
	assert.Equal("/var/dropdead", conf.DbPath, "DbPath should load correctly.")
	assert.Equal("/var/dropdead", conf.UploadsPath, "UploadsPath should load correctly.")
}
