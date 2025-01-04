package main

import (
	"github.com/samber/lo"
)

func main() {
	lo.Map([]string{}, func(a string, _ int) string { return "" })
}
