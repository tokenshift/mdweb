# Mdweb

Markdown-based language agnostic literate programming.

## Implementation

	<<#-->>
	package mdweb

	import "bufio"
	import "fmt"
	import "io"
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

File processing is implemented as a state machine. Each input line is an event,
and produces both a state transition and a single output `Line`.

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

Each line that is processed determines the next state.

	type lineType int
	const (
		Code lineType = iota
		Boilerplate
		Example
		Text
	)

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

The state machine starts out in the `Text` state.
	
	var state = Text

In addition to the state itself, the file processor keeps track of the current
output files. These are are initialized once, when the file processor starts.

	var defaultCodeOutput string
	var defaultTextOutput string
	var currentTarget string

Each line is processed by first checking for and handling a directive, and then
handling code, text or blank lines based on the current state.

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

While processing a code block, any blank lines are treated as part of the code.

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

Boilerplate is routed only to the code output, not to documentation. This state
will have been set once using a directive, and remains in effect as long as
code (indented) lines are being processed.

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

Example code is written _only_ to documentation, not to output code. Again,
this state will have been set using a directive and will persist until the next
non-code line.

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

When processing normal text, the only possible transitions are Text -> Text
(continue processing normal text) and Text -> Code (begin processing a code
block). The only way to switch to the Boilerplate or Example state is through
a directive.

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

There are 3 (and a half?) types of directives:

	func processDirective(directive string) {
		switch directive {

The `<<!-->>` directive indicates comment/example code, which should be
included in documentation, but not in code.

		case "!--":
			state = Example

The `<<#-->>` directive does the opposite, indicating boilerplate code that is
needed for the code output, but shouldn't be included in documentation.

		case "#--":
			state = Boilerplate

Any other directive is treated as a filename, to which all subsequent code will
be written. The 'half' case is an empty directive (e.g. `<<>>`), which simply
resets the target file to the default.

		default:
			state = Code
			if directive == "" {
				currentTarget = defaultCodeOutput
			} else {
				currentTarget = directive
			}
		}
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
