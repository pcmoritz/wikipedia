package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
)

// var inputFile = flag.String("infile", "enwiki-latest-pages-articles.xml", "Input file path")
var printLex = flag.Bool("print-lex", false, "Print output from lexer")

func parseBracket(l *lexer, left itemType, right itemType) {
	depth := 1
	for s := l.nextItem(); s.typ != itemEOF; s = l.nextItem() {
		if s.typ == left {
			depth += 1
		}
		if s.typ == right {
			depth -= 1
		}
		if depth == 0 {
			return
		}
	}
}

func parseLink(l *lexer) []item {
	text := make([]item, 0, 10)
	for s := l.nextItem(); s.typ != itemEOF; s = l.nextItem() {
		text = append(text, s)
		if s.typ == itemMark && s.val == "|" {
			text = text[0:0]
		}
		if s.typ == itemRightTag {
			break
		}
	}
	return text
}

func parseTitle(l *lexer, level int) []item {
	result := make([]item, 0, 10)
	for s := l.nextItem(); s.typ != itemEOF; s = l.nextItem() {
		result = append(result, s)
		if s.typ == itemTitle {
			break
		}
	}
	return result
}

func printElement(elt item) {
	if elt.typ == itemWord || elt.typ == itemSpace || elt.typ == itemMark {
		fmt.Print(elt.val)
	}
}

func main() {
	file, err := os.Open("article.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := scanner.Text()
		lexer := lex(str)
		// lexer = lex("<ref name=\"Best\"/> name")
		count := 0
		for s := lexer.nextItem(); s.typ != itemEOF; s = lexer.nextItem() {
			if s.typ == itemLeftMeta {
				parseBracket(lexer, itemLeftMeta, itemRightMeta)
			} else if s.typ == itemLeftTag {
				for _, s := range parseLink(lexer) {
					count += 1
					if *printLex {
						fmt.Print("(", s.typ, " ")
						fmt.Print(s.val, ")  ")
					} else {
						printElement(s)
					}
				}
			} else if s.typ == itemTitle {
				fmt.Println()
				for _, s := range parseTitle(lexer, len(s.val)) {
					printElement(s)
				}
				fmt.Println()
			} else {
				count += 1
				if *printLex {
					fmt.Print("(", s.typ, " ")
					fmt.Print(s.val, ")  ")
				} else {
					printElement(s)
				}
			}
		}
		fmt.Println("count ", count)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
