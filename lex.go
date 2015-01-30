package scribeconf

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// item represents a token returned from the scanner.
type item struct {
	typ itemType // Type, such as itemKey.
	val string   // Value, such as "category".
}

func (i item) String() string {
	switch i.typ {
	case itemError:
		return i.val
	case itemEOF:
		return "EOF"
	case itemKey:
		return fmt.Sprintf("Key: %s", i.val)
	case itemEqual:
		return "="
	case itemVal:
		return fmt.Sprintf("Val: %s", i.val)
	case itemLeftMeta:
		return fmt.Sprintf("%s", i.val)
	case itemRightMeta:
		return fmt.Sprintf("%s", i.val)
	default:
		return "unknown"
	}
}

// itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred, value is text of error.
	itemEOF

	itemKey       // key name of a field
	itemEqual     // the '=' sign
	itemVal       // value of a field
	itemLeftMeta  // the opening token of a block, for example "<store>"
	itemRightMeta // the closing token of a block, for example "</store>"
)

type lexer struct {
	input string    // the string being scanned.
	start int       // start position of this item.
	pos   int       // current position in the input,
	width int       // width of last rune read from input.
	items chan item // channel of scanned items.
}

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// run lexes the input by executing state functions until state is nil.
func (l *lexer) run() {
	for state := lexBlock; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// next returns the next rune in the input.
func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune. Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next run if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// errorf returns an error token and terminates the scan by passing back a nil pointer that will be the next state.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, fmt.Sprintf(format, args...)}
	return nil
}

func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrently run state machine.
	return l
}

const eof = -1

func lexBlock(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		l.emit(itemEOF)
		return nil
	case isSpace(r) || isEndOfLine(r):
		l.ignore()
	case r == '<':
		return lexMeta
	case r == '#':
		return lexComment
	case unicode.IsLetter(r):
		l.backup()
		return lexKey
	default:
		return l.errorf("unrecognized character in action: %#U", r)
	}
	return lexBlock
}

func lexMeta(l *lexer) stateFn {
	l.accept("/") // consume '/' if it is a right meta tag.
	for {
		r := l.next()
		if !isAlphaNumeric(r) {
			if r != '>' {
				return l.errorf("unclosed meta tag %#U", r)
			}
			break
		}
	}

	if l.input[l.start+1] == '/' {
		l.emit(itemRightMeta)
	} else {
		l.emit(itemLeftMeta)
	}
	return lexBlock
}

func lexComment(l *lexer) stateFn {
	for {
		r := l.next()
		if isEndOfLine(r) || r == eof {
			break
		}
	}
	l.ignore()
	return lexBlock
}

func lexKey(l *lexer) stateFn {
	for {
		r := l.next()
		if !isAlphaNumeric(r) {
			l.backup()
			break
		}
	}
	l.emit(itemKey)
	return lexEqual
}

func lexEqual(l *lexer) stateFn {
	eqNum := 0
	for {
		r := l.next()
		if r == '=' {
			eqNum += 1
		} else if isSpace(r) {
			// white space consumed
		} else {
			l.backup()
			break
		}
	}
	if eqNum != 1 {
		return l.errorf("error equal symbol occurrence: %d", eqNum)
	}

	l.emit(itemEqual)
	return lexVal
}

func lexVal(l *lexer) stateFn {
	for {
		r := l.next()
		if isEndOfLine(r) || r == eof {
			break
		}
	}
	l.emit(itemVal)
	return lexBlock
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
