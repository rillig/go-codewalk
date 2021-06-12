package main

// Lexer provides a flexible way of splitting a string into several parts
// by repeatedly chopping off a prefix that matches a string, a function
// or a set of byte values.
//
// The Next* methods chop off and return the matched portion.
//
// The Skip* methods chop off the matched portion and return whether something
// matched.
//
// PeekByte and TestByteSet look at the next byte without chopping it off.
// They are typically used in switch statements, which don't allow variable
// declarations.
type Lexer struct {
	rest string
}

// ByteSet is a subset of all 256 possible byte values.
// It is used for matching byte strings efficiently.
//
// It cannot match Unicode code points individually and is therefore
// usually used with ASCII characters.
type ByteSet struct {
	bits [256]bool
}

func NewLexer(text string) *Lexer {
	return &Lexer{text}
}

// Rest returns the part of the string that has not yet been chopped off.
func (l *Lexer) Rest() string { return l.rest }

// SkipHspace chops off the longest prefix (possibly empty) consisting
// solely of horizontal whitespace, which is the ASCII space (U+0020)
// and tab (U+0009).
func (l *Lexer) SkipHspace() bool {
	i := 0
	rest := l.rest
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	if i > 0 {
		l.rest = rest[i:]
		return true
	}
	return false
}

// NextBytesSet chops off the longest prefix (possibly empty) consisting
// solely of bytes from the given set.
func (l *Lexer) NextBytesSet(bytes *ByteSet) string {
	i := 0
	rest := l.rest
	for i < len(rest) && bytes.Contains(rest[i]) {
		i++
	}
	if i != 0 {
		l.rest = rest[i:]
	}
	return rest[:i]
}

// NewByteSet creates a bit mask out of a string like "0-9A-Za-z_".
// To add an actual hyphen to the bit mask, write it either at the beginning
// or at the end of the string, or directly after a range like "a-z".
//
// The bit mask can be used with Lexer.NextBytesSet.
func NewByteSet(chars string) *ByteSet {
	var set ByteSet
	i := 0

	for i < len(chars) {
		switch {
		case i+2 < len(chars) && chars[i+1] == '-':
			min := uint(chars[i])
			max := uint(chars[i+2]) // inclusive
			for c := min; c <= max; c++ {
				set.bits[c] = true
			}
			i += 3
		default:
			set.bits[chars[i]] = true
			i++
		}
	}
	return &set
}

// Contains tests whether the byte set contains the given byte.
func (bs *ByteSet) Contains(b byte) bool { return bs.bits[b] }
