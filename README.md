# mdweb

Markdown-based literate programming tools (mdtangle and mdweave).

## Use

**mdweb** processes Markdown files, separating them into text and code. The `mdtangle` command extracts code from a Markdown file and writes it to individual source files; the `mdweave` command does the reverse, writing the textual content of a Markdown file with all **mdweb** directives removed.

Arguments to both commands are a pattern to match filenames (e.g. "*.go.md" will match any `.go.md` file).

## Default Output

By default, `mdtangle` writes code to a file with the same name as the input with the last extension removed; `mdweave` writes to a file with the same name as the input with all extensions removed, and `.md` appended.

## Directives

A directive is a line of the form `<<...>>`, and must be placed on its own line in a code block (though it does not have to be at the beginning). With a file name (e.g. `<<target.go>>`), all subsequent code content will be written to that file. An empty/blank directive resets the target file to the default (see above). The `<<!-->>` directive is a special case for code blocks that should be included in the text output, but should not be considered actual code (like examples of use).

See https://raw.githubusercontent.com/tokenshift/mdweb/master/example.foo.md for an example input file that demonstrates the directives.
