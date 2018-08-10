package main

import (
	"fmt"
	"strings"
	"testing"
)

func Test_others(t *testing.T) {
	fp := func(str string) int {
		return strings.LastIndex(str[0:strings.LastIndex(str, ".")], ".")
	}
	str := "est.maple.casxt.com"
	fmt.Println(str[0:fp(str)], str[fp(str)+1:len(str)])
}
