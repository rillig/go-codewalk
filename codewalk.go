package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"netbsd.org/pkglint/textproc"
	"os"
	"regexp"
	"strings"
)

type block struct {
	lines    []string
	codewalk *codewalk
}

type codewalk struct {
	file  string
	start int // inclusive
	end   int // inclusive
}

func newCodewalk() *codewalk { return &codewalk{"", -1, -1} }

func (c *codewalk) setStart(re *regexp.Regexp) error {
	text, err := ioutil.ReadFile(c.file)
	lines := strings.Split(string(text), "\n")
	if err != nil {
		return err
	}

	count := 0
	for i, line := range lines {
		if re.MatchString(line) {
			c.start = i
			count++
		}
	}

	switch count {
	case 0:
		return fmt.Errorf("regular expression %q not found in %q", re, c.file)
	case 1:
		return nil
	default:
		return fmt.Errorf("regular expression %q found %d times in %q", re, count, c.file)
	}
}

func (c *codewalk) setEnd(re *regexp.Regexp) error {
	if c.start == 0 {
		return fmt.Errorf("the \"end\" command is only valid after a preceding \"start\" command")
	}

	text, err := ioutil.ReadFile(c.file)
	lines := strings.Split(string(text), "\n")
	if err != nil {
		return err
	}

	for i, line := range lines {
		if i > c.start && re.MatchString(line) {
			c.end = i
			return nil
		}
	}

	return fmt.Errorf("regular expression %q not found after %q:%d", re, c.file, c.start)
}

func (c *codewalk) setGoFunc(name string, doc bool, body bool) error {
	text, err := ioutil.ReadFile(c.file)
	lines := strings.Split(string(text), "\n")
	if err != nil {
		return err
	}

	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '.' })
	for i, part := range parts {
		parts[i] = "\\b\\Q" + part + "\\E\\b"
	}
	re := regexp.MustCompile("^func\\b.*" + strings.Join(parts, ".*")) // just a little cheated
	for i, line := range lines {
		if re.MatchString(line) {
			c.start = i
			if strings.HasSuffix(line, "{") {
				for j := i + 1; j < len(lines); j++ {
					if strings.HasPrefix(lines[j], "}") { // also cheated
						c.end = j

						for doc && c.start > 0 && strings.HasPrefix(lines[c.start-1], "//") {
							c.start--
						}

						return nil
					}
				}
			} else {
				c.end = c.start
				return nil
			}
			return fmt.Errorf("end of function %q not found after %s:%d", name, c.file, c.start)
		}
	}

	return fmt.Errorf("function %q not found in %q", name, c.file)
}

func (c *codewalk) setGoType(name string, doc bool, body bool) error {
	text, err := ioutil.ReadFile(c.file)
	lines := strings.Split(string(text), "\n")
	if err != nil {
		return err
	}

	needle := "type " + name + " "
	for i, line := range lines {
		if i > c.start && strings.HasPrefix(line, needle) {
			c.start = i
			if strings.Contains(line, "struct {") {
				for j := i + 1; j < len(lines); j++ {
					if strings.HasPrefix(lines[j], "}") { // also cheated
						c.end = j

						for doc && c.start > 0 && strings.HasPrefix(lines[c.start-1], "//") {
							c.start--
						}

						return nil
					}
				}
			} else {
				c.end = c.start
				return nil
			}
			return fmt.Errorf("end of type %q not found after %s:%d", name, c.file, c.start)
		}
	}

	return fmt.Errorf("type %q not found in %q", name, c.file)
}

func (c *codewalk) finish() ([]string, error) {
	text, err := ioutil.ReadFile(c.file)
	lines := strings.Split(string(text), "\n")
	if err != nil {
		return nil, err
	}

	if c.start == -1 {
		return nil, fmt.Errorf("missing start for codewalk block")
	}
	if c.end == -1 {
		return nil, fmt.Errorf("missing end for codewalk block")
	}

	return lines[c.start : c.end+1], nil
}

func GenerateCodewalk(src string, dst string, basedir string) error {
	srcText, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	var blocks []*block
	curr := &block{}

	emit := func() {
		if len(curr.lines) > 0 {
			blocks = append(blocks, curr)
			curr = &block{}
		}
	}

	lineno := 0
	for _, line := range strings.Split(string(srcText), "\n") {
		lineno++
		switch {
		case curr.codewalk == nil && line == "```codewalk":
			emit()
			curr.codewalk = newCodewalk()

		case curr.codewalk != nil && line == "```":
			code, err := curr.codewalk.finish()
			if err != nil {
				return err
			}
			curr.lines = code
			emit()

		case curr.codewalk == nil:
			curr.lines = append(curr.lines, line)

		default:
			lex := textproc.NewLexer(line)
			cmd := lex.NextBytesSet(textproc.NewByteSet("A-Za-z:"))
			lex.SkipHspace()

			switch cmd {
			case "file":
				curr.codewalk.file = lex.Rest()

			case "start":
				err = curr.codewalk.setStart(regexp.MustCompile(lex.Rest()))
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			case "end":
				err = curr.codewalk.setEnd(regexp.MustCompile(lex.Rest()))
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			case "go:func":
				doc := true
				body := true

				args := strings.Fields(lex.Rest())
				for len(args) > 0 {
					if args[0] == "-no-doc" {
						doc = true
					} else if args[0] == "-no-body" {
						body = false
					} else {
						break
					}
				}
				err = curr.codewalk.setGoFunc(lex.Rest(), doc, body)
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			case "go:type":
				doc := true
				body := true

				args := strings.Fields(lex.Rest())
				for len(args) > 0 {
					if args[0] == "-no-doc" {
						doc = true
					} else if args[0] == "-no-body" {
						body = false
					} else {
						break
					}
				}
				err = curr.codewalk.setGoType(lex.Rest(), doc, body)
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			default:
				return fmt.Errorf("%s:%d: invalid codewalk command %q", src, lineno, cmd)
			}
		}
	}

	var lines []string

	for _, block := range blocks {
		if block.codewalk != nil {
			lines = append(lines,
				fmt.Sprintf("> from [%[1]s](%[1]s#L%d):", block.codewalk.file, block.codewalk.start),
				"",
				"```go")
			lines = append(lines, block.lines...)
			lines = append(lines, "```")
		} else {
			lines = append(lines, block.lines...)
		}
	}

	err = ioutil.WriteFile(dst, []byte(strings.Join(append(lines, ""), "\n")), 0666)
	return err
}

func main() {
	if len(os.Args) != 4 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <source.md> <target.md> <basedir>\n", os.Args[0])
		os.Exit(1)
	}

	err := GenerateCodewalk(os.Args[1], os.Args[2], os.Args[3])
	if err != nil {
		log.Fatal(err)
	}
}
