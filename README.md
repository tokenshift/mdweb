# mdweb

Markdown-based literate programming tools (mdtangle and mdweave).

## Use

**mdweb** processes Markdown files, separating them into text and code. The
`mdtangle` command extracts code from a Markdown file and writes it to
individual source files; the `mdweave` command does the reverse, writing the
textual content of a Markdown file with all **mdweb** directives removed.

Arguments to both commands are a pattern to match filenames (e.g. "*.go.md"
will match any `.go.md` file).

### Default Output

By default, `mdtangle` writes code to a file with the same name as the input
with the last extension removed; `mdweave` writes to a file with the same name
as the input with all extensions removed, and `.md` appended.

### Directives

A directive is a line of the form `<<...>>`, and must be placed on its own line
in a code block (though it does not have to be at the beginning of the block).

* `<<filename>>`  
  Sets the target file to the specified (relative) path. All subsequent code
  will be written to that file, rather than the default.
* `<<>>`  
  Resets the target file to the default.
* `<<!-->>`  
  Example code that should be included in text output, but not in code.
* `<<#-->>`  
  Boilerplate code that should be omitted from the text output, but included in
  the code.

See https://raw.githubusercontent.com/tokenshift/mdweb/master/example.foo.md
for an example input file that demonstrates the directives.

## Installation

    go get github.com/tokenshift/mdweb
    go build github.com/tokenshift/mdweb/mdtangle
    go build github.com/tokenshift/mdweb/mdweave
    go install github.com/tokenshift/mdweb/mdtangle
    go install github.com/tokenshift/mdweb/mdweave
