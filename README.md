# Codewalk

The Go standard library provides so-called codewalks, which are navigable
documents that refer to source code of the project and serve as an
introduction to the code.

Unfortunately they are limited to the Go standard library and cannot be
used for third-party code. Therefore I wrote this alternative, using Markdown
as the primary markup language instead of XML.

There are two documents in this folder: [README.src.md](README.src.md)
contains the source code of the codewalk, while [README.md](README.md)
contains the documentation generated from that file. It is generated
using this command:

~~~shell script
codewalk README.src.md README.md
~~~

To include a code snippet in the document, add a block like the following:

    ```codewalk
    file    codewalk.go
    start   ^func Name
    end     ^}
    endUp   5
    go:func FunctionName
    go:func -no-doc -no-body Type.Method
    go:type -no-doc -no-body Type
    ```

Each line in the above block is a command. The above commands are just a
summary and cannot all be combined. To get an idea about the possibilities,
here is the type definition of a code block:

> from [codewalk.go](codewalk.go#L15):

```go
type block struct {
	lines   []string
	snippet *snippet
}
```

The lines of a block are contiguous, though it would be possible to add
some "grep" feature, to only show matching lines.

When a block is parsed, its `snippet` field gets filled. Its definition is:

> from [codewalk.go](codewalk.go#L20):

```go
type snippet struct {
	file  string
	start int // inclusive
	end   int // inclusive
}
```

Here it becomes obvious that the commands of a `codewalk` block in the
Markdown document can only manipulate the start and end line of the block.

To see a more complete example file, have a look at the codewalk of the
[pkglint](https://github.com/rillig/pkglint/blob/master/v23/codewalk.md) project.
