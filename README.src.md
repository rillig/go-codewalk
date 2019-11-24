# Codewalk

The Go standard library provides so-called codewalks, which are navigable
documents that refer to source code of the project and serve as an
introduction to the code.

Unfortunately they are limited to the Go standard library and cannot be
used for third-party code. Therefore I wrote this alternative, using Markdown
as the primary markup language instead of XML.

There are two documents in this folder: [codewalk.src.go] contains the
source code of the codewalk, while [codewalk.md] contains the documentation
generated from that file. It is generated using this command:

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

```codewalk
file    codewalk.go
go:type block
```

The lines of a block are contiguous, though it would be possible to add
some "grep" feature, to only show matching lines.

When a block is parsed, its `codewalk` field gets filled. Its definition is:

```codewalk
file    codewalk.go
go:type codewalk
```

Here it becomes obvious that the commands of a `codewalk` block in the
Markdown document can only manipulate the start and end line of the block.

To see a more complete example file, have a look at the codewalk of the
[pkglint](https://github.com/rillig/pkglint/blob/master/codewalk.md) project.
