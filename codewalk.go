package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
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
			if strings.HasSuffix(line, "{") && body {
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
			if strings.Contains(line, "struct {") && body {
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
				for doc && c.start > 0 && strings.HasPrefix(lines[c.start-1], "//") {
					c.start--
				}
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

func GenerateCodewalk(src string, dst string) error {
	srcBytes, err := ioutil.ReadFile(src)
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
	srcText := strings.Replace(string(srcBytes), "\r\n", "\n", -1)
	srcText = strings.TrimRight(srcText, "\n")
	srcLines := strings.Split(srcText, "\n")
	for _, line := range srcLines {
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
			lex := NewLexer(line)
			cmd := lex.NextBytesSet(NewByteSet("A-Za-z:"))
			lex.SkipHspace()

			switch cmd {
			case "file":
				curr.codewalk.file = lex.Rest()

			case "start":
				re, err := regexp.Compile(lex.Rest())
				if err == nil {
					err = curr.codewalk.setStart(re)
				}
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			case "end":
				re, err := regexp.Compile(lex.Rest())
				if err == nil {
					err = curr.codewalk.setEnd(re)
				}
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			case "endUp":
				n, err := strconv.Atoi(lex.Rest())
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}
				curr.codewalk.end -= n

			case "go:func":
				flags := flag.NewFlagSet("go:func", flag.ContinueOnError)
				noDoc := flags.Bool("no-doc", false, "")
				noBody := flags.Bool("no-body", false, "")

				err := flags.Parse(strings.Fields(lex.Rest()))
				if err != nil || len(flags.Args()) != 1 {
					return fmt.Errorf("%s:%d: usage: go:func [-no-doc] [-no-body] [<Type>.]<Name>", src, lineno)
				}

				err = curr.codewalk.setGoFunc(flags.Arg(0), !*noDoc, !*noBody)
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			case "go:type":
				flags := flag.NewFlagSet("go:type", flag.ContinueOnError)
				noDoc := flags.Bool("no-doc", false, "")
				noBody := flags.Bool("no-body", false, "")

				err := flags.Parse(strings.Fields(lex.Rest()))
				if err != nil || len(flags.Args()) != 1 {
					return fmt.Errorf("%s:%d: usage: go:type [-no-doc] [-no-body] <Type>", src, lineno)
				}

				err = curr.codewalk.setGoType(flags.Arg(0), !*noDoc, !*noBody)
				if err != nil {
					return fmt.Errorf("%s:%d: %s", src, lineno, err)
				}

			default:
				return fmt.Errorf("%s:%d: invalid codewalk command %q", src, lineno, cmd)
			}
		}
	}
	emit()

	var lines []string

	for _, block := range blocks {
		if block.codewalk != nil {
			lines = append(lines,
				fmt.Sprintf("> from [%[1]s](%[1]s#L%d):", block.codewalk.file, block.codewalk.start+1),
				"",
				"```go")
			lines = append(lines, block.lines...)
			lines = append(lines, "```")
		} else {
			lines = append(lines, block.lines...)
		}
	}

	var sb bytes.Buffer
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	err = ioutil.WriteFile(dst, sb.Bytes(), 0666)
	return err
}

func main() {
	if len(os.Args) != 3 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s <source.md> <target.md>\n", os.Args[0])
		os.Exit(1)
	}

	err := GenerateCodewalk(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
}
