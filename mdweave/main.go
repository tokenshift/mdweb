package main

import "fmt"
import "os"
import "path/filepath"

import "github.com/tokenshift/mdweb"

func main() {
	// Each argument is treated as a glob specification.
	for _, arg := range os.Args[1:] {
		files, _ := filepath.Glob(arg)
		for _, file := range files {
			target, lines, err := mdweb.ProcessFile(file)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			file, err := os.Create(target)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			fmt.Println("Writing text to", target)
			for line := range lines {
				if line.TextTarget == "" {
					continue
				}

				if line.CodeTarget != "" {
					fmt.Fprint(file, "\t")
				}

				fmt.Fprint(file, line.Text)
			}
		}
	}
}
