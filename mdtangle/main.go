package main

import "fmt"
import "os"
import "path/filepath"

import "github.com/tokenshift/mdweb"

func main() {
	// Keep track of files that are open for writing.
	outputFiles := make(map[string]*os.File)

	// Each argument is treated as a glob specification.
	for _, arg := range os.Args[1:] {
		files, _ := filepath.Glob(arg)
		for _, file := range files {
			_, lines, err := mdweb.ProcessFile(file)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			for line := range lines {
				if line.CodeTarget != "" {
					out, ok := outputFiles[line.CodeTarget]
					if !ok {
						out, err = os.Create(line.CodeTarget)
						if err != nil {
							fmt.Fprintln(os.Stderr, err)
							os.Exit(1)
						}
						defer out.Close()
						fmt.Println("Writing code to", line.CodeTarget)
						outputFiles[line.CodeTarget] = out
					}

					fmt.Fprint(out, line.Text)
				}
			}
		}
	}
}
