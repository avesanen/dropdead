package main

import (
	"math/rand"
)

var (
	rndNameLength = 6
	rndNameRunes  = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func RandomName() string {
	b := make([]rune, rndNameLength)
	for i := range b {
		b[i] = rndNameRunes[rand.Intn(len(rndNameRunes))]
	}
	return string(b)
}
