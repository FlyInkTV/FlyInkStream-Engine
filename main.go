// main executable.
package main

import (
	"os"

	"github.com/FlyInkTV/FlyInkStream-Engine/internal/core"
)

func main() {
	s, ok := core.New(os.Args[1:])
	if !ok {
		os.Exit(1)
	}
	s.Wait()
}




