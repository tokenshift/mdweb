package mdweb

import "bufio"
import "fmt"
import "io"
import "os"
import "path/filepath"
import "regexp"
import "strings"

type Line struct {
	Code string
	CodeTarget string
	Text string
	TextTarget string
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

func ProcessFile(filename string) (lines <-chan Line, err error) {
	defaultCodeOutput = removeExtension(filename)
	defaultTextOutput = removeExtensions(filename) + ".md"
	currentTarget = defaultCodeOutput

	input, err := os.Open(filename)
	if err != nil {
		return
	}

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

	return out, nil
}

type lineType int
const (
	Code lineType = iota
	Boilerplate
	Example
	Text
)

var rxDirective = regexp.MustCompile("^<<(.*)>>\\s*$")

func unindent(line string) (string, bool) {
	if strings.HasPrefix(line, "\t") {
		return line[1:], true
	} else if strings.HasPrefix(line, "    ") {
		return line[4:], true
	} else {
		return line, false
	}
}

func parseDirective(line string) (directive string, ok bool) {
	line, isIndented := unindent(line)
	if !isIndented {
		return "", false
	}

	matches := rxDirective.FindStringSubmatch(line)
	if matches == nil {
		return "", false
	}

	return strings.TrimSpace(matches[1]), true
}


var state = Text

var defaultCodeOutput string
var defaultTextOutput string
var currentTarget string

func processLine(line string, lines chan<- Line) {
	directive, isDirective := parseDirective(line)
	if isDirective {
		processDirective(directive)
		return
	}

	codeLine, isCode := unindent(line)
	isBlank := strings.TrimSpace(line) == ""

	switch state {
	case Code:

		if isCode || isBlank {
			lines <- Line {
				Code: codeLine,
				CodeTarget: currentTarget,
				Text: line,
				TextTarget: defaultTextOutput,
			}
		} else {
			state = Text
			lines <- Line {
				Code: "",
				CodeTarget: "",
				Text: line,
				TextTarget: defaultTextOutput,
			}
		}

	case Boilerplate:
		if isCode || isBlank {
			lines <- Line {
				Code: codeLine,
				CodeTarget: currentTarget,
				Text: "",
				TextTarget: "",
			}
		} else {
			state = Text
			lines <- Line {
				Code: "",
				CodeTarget: "",
				Text: line,
				TextTarget: defaultTextOutput,
			}
		}

	case Example:
		if isCode || isBlank {
			lines <- Line {
				Code: "",
				CodeTarget: "",
				Text: line,
				TextTarget: defaultTextOutput,
			}
		} else {
			state = Text
			lines <- Line {
				Code: "",
				CodeTarget: "",
				Text: line,
				TextTarget: defaultTextOutput,
			}
		}

	case Text:
		if isCode {
			state = Code
			lines <- Line {
				Code: codeLine,
				CodeTarget: currentTarget,
				Text: line,
				TextTarget: defaultTextOutput,
			}
		} else {
			lines <- Line {
				Code: "",
				CodeTarget: "",
				Text: line,
				TextTarget: defaultTextOutput,
			}
		}
	}
}

func processDirective(directive string) {
	switch directive {

	case "!--":
		state = Example

	case "#--":
		state = Boilerplate

	default:
		state = Code
		if directive == "" {
			currentTarget = defaultCodeOutput
		} else {
			currentTarget = directive
		}
	}
}

func ProcessFiles(writeCode, writeText bool, patterns ...string) {
	outputFiles := make(map[string]*os.File)

	for _, pattern := range patterns {
		files, _ := filepath.Glob(pattern)
		for _, file := range files {
			lines, err := ProcessFile(file)

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			for line := range lines {
				if writeCode && line.CodeTarget != "" {
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

					fmt.Fprint(out, line.Code)
				}

				if writeText && line.TextTarget != "" {
					out, ok := outputFiles[line.TextTarget]
					if !ok {
						out, err = os.Create(line.TextTarget)
						if err != nil {
							fmt.Fprintln(os.Stderr, err)
							os.Exit(1)
						}
						defer out.Close()
						fmt.Println("Writing documentation to", line.TextTarget)
						outputFiles[line.TextTarget] = out
					}

					fmt.Fprint(out, line.Text)
				}
			}
		}
	}
}
