// A lexer for xml dumps from wikipedia
// Inspired by Rob Pike's lexer for go templates
// (c) Philipp Moritz, 2014

package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const eof = -1

// stateFn represents the state of the scanner as a function that
// returns the next state.
type stateFn func(*lexer) stateFn

type lexer struct {
	input string    // the string being scanned.
	state stateFn   // the next lexing function to enter.
	start int       // start position of this item.
	pos   int       // current position in the input.
	width int       // width of last rune read from input.
	items chan item // channel of scanned items.
}

type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemLeftMeta
	itemRightMeta
	itemLeftTag
	itemRightTag
	itemNumber
	itemWord
	itemQuote
	itemSpace
	itemMark
	itemXML
	itemTitle
)

type item struct {
	typ itemType
	val string
}

// lex creates a new scanner for the input string.
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		state: lexArticle,
		items: make(chan item),
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer
func (l *lexer) run() {
	for l.state = lexArticle; l.state != nil; {
		l.state = l.state(l)
	}
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	return item
}

// error returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

// next returns the next rune in the input
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune. Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	rune := l.next()
	l.backup()
	return rune
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func lexNumber(l *lexer) stateFn {
	digits := "0123456789"
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	l.emit(itemNumber)
	return lexArticle
}

func lexArticle(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		l.emit(itemEOF)
		return nil
	case r == '{' && l.peek() == '{':
		l.next()
		l.emit(itemLeftMeta)
		return lexArticle
	case r == '}' && l.peek() == '}':
		l.next()
		l.emit(itemRightMeta)
		return lexArticle
	case r == '[' && l.peek() == '[':
		l.next()
		l.emit(itemLeftTag)
		return lexArticle
	case r == ']' && l.peek() == ']':
		l.next()
		l.emit(itemRightTag)
		return lexArticle
	case r == '\'':
		return lexQuote
	case r == '<':
		l.backup()
		return lexXML
	case r == '=':
		return lexTitle
	case isSpace(r):
		return lexSpace
	case unicode.IsMark(r) || unicode.IsSymbol(r) || unicode.IsPunct(r):
		l.emit(itemMark)
		return lexArticle
	case isAlphaNumeric(r):
		return lexWord
	}
	return lexArticle
}

// lexSpace scans a run of space characters. One space has already been seen.
func lexSpace(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
	}
	l.emit(itemSpace)
	return lexArticle
}

func lexWord(l *lexer) stateFn {
	for {
		r := l.next()
		if r == '\'' && l.peek() == '\'' {
			l.backup()
			l.emit(itemWord)
			break
		}
		if isAlphaNumeric(r) || r == '-' || r == '\'' {
			// absorb
		} else {
			l.backup()
			l.emit(itemWord)
			break
		}
	}
	return lexArticle
}

func lexQuote(l *lexer) stateFn {
	for l.peek() == '\'' {
		l.next()
	}
	l.emit(itemQuote)
	return lexArticle
}

func lexTitle(l *lexer) stateFn {
	for l.peek() == '=' {
		l.next()
	}
	l.emit(itemTitle)
	return lexArticle
}

func lexXML(l *lexer) stateFn {
	reader := strings.NewReader(l.input[l.pos:])
	u := reader.Len()
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	_, err := decoder.RawToken()
	if err != nil {
		fmt.Println(err)
		panic("Error during parsing XML.")
	}
	v := reader.Len()
	l.pos += u - v
	l.emit(itemXML)
	return lexArticle
}
