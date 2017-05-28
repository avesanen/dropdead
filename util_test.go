package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomName(t *testing.T) {
	lenTests := []int{4, 7, 10}
	for _, n := range lenTests {
		rndNameLength = n
		rndName := RandomName()
		assert.Len(t, rndName, rndNameLength, fmt.Sprintf("Random name should be %d characters long.", rndNameLength))
	}
}
