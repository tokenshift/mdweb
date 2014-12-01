# Mdweb

Markdown-based language agnostic literate programming.

## Implementation

	<<#-->>
	package mdweb

	import "bufio"
	import "fmt"
	import "os"
	import "path/filepath"
	import "regexp"
	import "strings"

**Mdweb** processes a single file at a time, splitting it into a series of
lines to be written to an output code file, text/documentation, or both. Each
line is represented as a `Line` struct containing the text of the line and two
target files; if the current line should not be written to either output code
or documentation, that target field will be empty.

	type Line struct {
		Code string
		CodeTarget string
		Text string
		TextTarget string
	}

The output file names are selected by removing one or more extensions from the
input filename. Documentation is written to a file with all of the extensions
replaced with `.md`; code is written to a file with only the last extension
removed.

As an example, a file named `foo.cpp.md` will produce documentation at `foo.md`
and code at `foo.cpp`.

The output code filename can be overridden with the use of a _target directive_
(see below for more info).

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

File processing is implemented as a state machine. Each state is a function
that takes the next line of input and returns the next state, as well as
potentially producing a single output `Line`.

	type State func(data StateData, inputLine string) BoundState
	type BoundState func(inputLine string) BoundState

In addition to the state itself, the file processor keeps track of the current
output files.

	type StateData struct {
		DefaultCodeOutput string
		DefaultTextOutput string
		CurrentTarget string
		Output chan<- Line
	}

There are four different states, corresponding to the type of the last line
that was read:

* `Code`  
  Any text indented by a tab or 4 space.
* `Boilerplate`  
  Code that shouldn't be included in documentation.
* `Example`  
  Code that should _only_ be included in documentation.
* `Text`  
  Any other text.

While processing a code block, any blank lines are treated as part of the code.

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

Boilerplate is routed only to the code output, not to documentation. This state
will have been set once using a directive, and remains in effect as long as
code (indented) lines are being processed.

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

Example code is written _only_ to documentation, not to output code. Again,
this state will have been set using a directive and will persist until the next
non-code line.

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

When processing normal text, the only possible transitions are Text -> Text
(continue processing normal text) and Text -> Code (begin processing a code
block). The only way to switch to the Boilerplate or Example state is through
a directive.

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
	
Two additional line types are recognized, but not used as states. Directives
are lines of the form `<<`_`directive`_`>>`, indented like code:

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

and blank lines, as a special case, may be treated as code, documentation or
both, depending on the current state.

Each line is processed by first checking for and handling a directive, and then
handling code, text or blank lines based on the current state.

	func partialState(s State, data StateData) BoundState {
		return func(inputLine string) BoundState {
			if directive, isDirective := parseDirective(inputLine); isDirective {
				return processDirective(data, directive)
			} else {
				return s(data, inputLine)
			}
		}
	}

There are 3 (and a half?) types of directives:

	func processDirective(data StateData, directive string) BoundState {
		switch directive {

The `<<!-->>` directive indicates comment/example code, which should be
included in documentation, but not in code.

		case "!--":
			return partialState(stateExample, data)

The `<<#-->>` directive does the opposite, indicating boilerplate code that is
needed for the code output, but shouldn't be included in documentation.

		case "#--":
			return partialState(stateBoilerplate, data)

Any other directive is treated as a filename, to which all subsequent code will
be written. The 'half' case is an empty directive (e.g. `<<>>`), which simply
resets the target file to the default.

		default:
			if directive == "" {
				data.CurrentTarget = data.DefaultCodeOutput
			} else {
				data.CurrentTarget = directive
			}
			return partialState(stateCode, data)
		}
	}

The output files are initialized when processing begins; only the
`CurrentTarget` may change during file processing.

	func ProcessFile(filename string) (lines <-chan Line, err error) {
		out := make(chan Line)

		defaultCodeOutput := removeExtension(filename)
		data := StateData {
			DefaultCodeOutput: defaultCodeOutput,
			DefaultTextOutput: removeExtensions(filename) + ".md",
			CurrentTarget: defaultCodeOutput,
			Output: out,
		}

The state machine begins in the Text state.

		currentState := partialState(stateText, data)

A single line at a time is read from the input stream and passed to the current
state function.

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

The command line tools `mdtangle` and `mdweave` take filesystem globs as
arguments and process each matching file. Because the contents for each output
file may be spread across multiple input files (and redirected using
directives), output files are kept open until all input files have been
processed.

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
							err = os.MkdirAll(filepath.Dir(absCodeTarget), 0700)
							if err != nil {
								fmt.Fprintln(os.Stderr, err)
								os.Exit(1)
							}

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
							err = os.MkdirAll(filepath.Dir(absTextTarget), 0700)
							if err != nil {
								fmt.Fprintln(os.Stderr, err)
								os.Exit(1)
							}

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

Filenames in target directives are relative to the literate source, rather than
the current working directory.

	func getAbsTargetPath(source, targetPath string) string {
		if filepath.IsAbs(targetPath) {
			return targetPath
		}

		sourceDir := filepath.Dir(source)
		path := filepath.Join(sourceDir, targetPath)
		abs, err := filepath.Abs(path)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return abs
	}
