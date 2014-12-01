package mdweb

import "bufio"
import "fmt"
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

type State func(data StateData, inputLine string) BoundState
type BoundState func(inputLine string) BoundState

type StateData struct {
	DefaultCodeOutput string
	DefaultTextOutput string
	CurrentTarget string
	Output chan<- Line
}

func stateCode(data StateData, inputLine string) BoundState {
	codeLine, isCode := unindent(inputLine)
	isBlank := strings.TrimSpace(inputLine) == ""

	if isCode || isBlank {
		data.Output <- Line {
			Code: codeLine,
			CodeTarget: data.CurrentTarget,
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateCode, data)
	} else {
		data.Output <- Line {
			Code: "",
			CodeTarget: "",
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateText, data)
	}
}

func stateBoilerplate(data StateData, inputLine string) BoundState {
	codeLine, isCode := unindent(inputLine)
	isBlank := strings.TrimSpace(inputLine) == ""

	if isCode || isBlank {
		data.Output <- Line {
			Code: codeLine,
			CodeTarget: data.CurrentTarget,
			Text: "",
			TextTarget: "",
		}
		return partialState(stateBoilerplate, data)
	} else {
		data.Output <- Line {
			Code: "",
			CodeTarget: "",
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateText, data)
	}
}

func stateExample(data StateData, inputLine string) BoundState {
	_, isCode := unindent(inputLine)
	isBlank := strings.TrimSpace(inputLine) == ""

	if isCode || isBlank {
		data.Output <- Line {
			Code: "",
			CodeTarget: "",
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateExample, data)
	} else {
		data.Output <- Line {
			Code: "",
			CodeTarget: "",
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateText, data)
	}
}

func stateText(data StateData, inputLine string) BoundState {
	codeLine, isCode := unindent(inputLine)

	if isCode {
		data.Output <- Line {
			Code: codeLine,
			CodeTarget: data.CurrentTarget,
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateCode, data)
	} else {
		data.Output <- Line {
			Code: "",
			CodeTarget: "",
			Text: inputLine,
			TextTarget: data.DefaultTextOutput,
		}
		return partialState(stateText, data)
	}
}

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

func partialState(s State, data StateData) BoundState {
	return func(inputLine string) BoundState {
		if directive, isDirective := parseDirective(inputLine); isDirective {
			return processDirective(data, directive)
		} else {
			return s(data, inputLine)
		}
	}
}

func processDirective(data StateData, directive string) BoundState {
	switch directive {

	case "!--":
		return partialState(stateExample, data)

	case "#--":
		return partialState(stateBoilerplate, data)

	default:
		if directive == "" {
			data.CurrentTarget = data.DefaultCodeOutput
		} else {
			data.CurrentTarget = directive
		}
		return partialState(stateCode, data)
	}
}

func ProcessFile(filename string) (lines <-chan Line, err error) {
	out := make(chan Line)

	defaultCodeOutput := removeExtension(filename)
	data := StateData {
		DefaultCodeOutput: defaultCodeOutput,
		DefaultTextOutput: removeExtensions(filename) + ".md",
		CurrentTarget: defaultCodeOutput,
		Output: out,
	}

	currentState := partialState(stateText, data)

	input, err := os.Open(filename)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(input)

	go func () {
		defer close(data.Output)
		defer input.Close()

		for scanner.Scan() {
			currentState = currentState(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()

	return out, nil
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
					absCodeTarget := getAbsTargetPath(file, line.CodeTarget)
					out, ok := outputFiles[absCodeTarget]
					if !ok {
						out, err = os.Create(absCodeTarget)
						if err != nil {
							fmt.Fprintln(os.Stderr, err)
							os.Exit(1)
						}
						defer out.Close()
						fmt.Println("Writing code to", absCodeTarget)
						outputFiles[absCodeTarget] = out
					}

					fmt.Fprintln(out, line.Code)
				}

				if writeText && line.TextTarget != "" {
					absTextTarget := getAbsTargetPath(file, line.TextTarget)
					out, ok := outputFiles[absTextTarget]
					if !ok {
						out, err = os.Create(absTextTarget)
						if err != nil {
							fmt.Fprintln(os.Stderr, err)
							os.Exit(1)
						}
						defer out.Close()
						fmt.Println("Writing documentation to", absTextTarget)
						outputFiles[absTextTarget] = out
					}

					fmt.Fprintln(out, line.Text)
				}
			}
		}
	}
}

func getAbsTargetPath(source, targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}

	sourceDir := filepath.Dir(source)
	path := filepath.Join(sourceDir, targetPath)
	abs, err := filepath.Abs(path)

	if err != nil {
		panic(err)
	}

	return abs
}
