package mdweb

import "bufio"
import "io"
import "os"
import "path/filepath"
import "regexp"
import "strings"

type Line struct {
	CodeTarget string
	TextTarget string
	Text string
}

// Splits an input file into code and text chunks. Code chunks may be prefixed
// with the desired output file, so that not all code from a single input file
// needs to go to the same output file.
func ProcessFile(filename string) (textTarget string, lines <-chan Line, err error) {
	input, err := os.Open(filename)
	if err != nil {
		return
	}

	// The default output code filename is the filename with the last extension
	// removed.
	targetCodeFile = removeExtension(filename)

	// The default output text filename is the base name of the file (with all
	// extensions removed), with the ".md" extension added.
	targetTextFile = removeExtensions(filename) + ".md"

	reader := bufio.NewReader(input)
	out := make(chan Line)

	go func () {
		defer close(out)

		for {
			line, err := reader.ReadString('\n')
			if err == nil || err == io.EOF {
				processLine(line, out)
			}

			if err != nil {
				break
			}
		}
	}()

	return targetTextFile, out, nil
}

var currentTarget string
var isBoilerplate = false
var isExample = false
var targetCodeFile string
var targetTextFile string
var rxDirective = regexp.MustCompile("^<<(.*)>>\\s*$")
var writingCode = false

// Figure out what to do with an individual line.
func processLine(line string, lines chan Line) {
	if currentTarget == "" {
		currentTarget = targetCodeFile
	}

	if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ") {
		// Process as code, unless part of a comment block.
		if isExample {
			lines <- Line {
				CodeTarget: "",
				TextTarget: targetTextFile,
				Text: line,
			}
			return
		}

		// Code; remove the whitespace prefix.
		if line[0] == '\t' {
			line = line[1:]
		} else {
			line = line[4:]
		}

		matches := rxDirective.FindStringSubmatch(line)
		if matches != nil {
			// Process the directive.
			directive := strings.TrimSpace(matches[1])
			switch directive {
			case "!--":
				// This code block is a comment. Treat it as text rather than
				// code.
				isExample = true
			case "#--":
				// This code block is boilerplate, which should be omitted from
				// the text output.
				isBoilerplate = true
			default:
				// Set the target file.
				currentTarget = directive
			}

			return
		}

		if !isExample {
			writingCode = true
		}

		// Otherwise, write to the current target code file.
		if isBoilerplate {
			lines <- Line {
				CodeTarget: currentTarget,
				TextTarget: "",
				Text: line,
			}
		} else {
			lines <- Line {
				CodeTarget: currentTarget,
				TextTarget: targetTextFile,
				Text: line,
			}
		}

		return
	}

	// If the line is blank and code was being written, include a blank line
	// in the last code block.
	if writingCode && strings.TrimSpace(line) == "" {
		// If boilerplate was being written, omit the blank line from the text
		// output.
		if isBoilerplate {
			lines <- Line {
				CodeTarget: currentTarget,
				TextTarget: "",
				Text: line,
			}
		} else {
			lines <- Line {
				CodeTarget: currentTarget,
				TextTarget: targetTextFile,
				Text: line,
			}
		}

		return
	}

	// Write all other text directly to the output text file.
	isBoilerplate = false
	isExample = false
	writingCode = false
	lines <- Line {
		CodeTarget: "",
		TextTarget: targetTextFile,
		Text: line,
	}
}

// Removes a single extension from the filename.
func removeExtension(filename string) string {
	filename = filepath.Base(filename)
	ext := filepath.Ext(filename)
	return filename[0:len(filename)-len(ext)]
}

// Removes all extensions from the filename.
func removeExtensions(filename string) string {
	f1, f2 := filename, removeExtension(filename)
	for f1 != f2 {
		f1, f2 = f2, removeExtension(f2)
	}

	return f2
}
