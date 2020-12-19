package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
)

type color string

const (
	BLACK   = color("black")
	RED     = color("red")
	GREEN   = color("green")
	YELLOW  = color("yellow")
	BLUE    = color("blue")
	MAGENTA = color("magenta")
	CYAN    = color("CYAN")
	WHITE   = color("white")

	RESET   = color("reset")
	OVERLAP = ("overlap")
)

type (
	colorMap map[color]string
	exprsMap map[color][]*regexp.Regexp

	segment struct {
		begin int
		end   int
		c     color
	}
)

var (
	colors = colorMap{
		"black":   "\033[30m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",

		"reset":   "\033[0m",
		"overlap": "\033[37;1;4m",
	}

	reset = "\033[m"
)

func newSegment(begin, end int, col color) *segment {
	return &segment{begin, end, col}
}

func (s *segment) contains(x int) bool {
	return s.begin <= x && x <= s.end
}

func (s *segment) String() string {
	return fmt.Sprintf("(%d, %d, %s)", s.begin, s.end, s.c)
}

func colorize(c color, s string) string {
	if colors[c] == "" {
		panic("Color not found")
	}

	return fmt.Sprintf("%s%s%s", colors[c], s, reset)
}

func partition(segments []*segment) []*segment {
	sort.Slice(segments, func(i, j int) bool { return segments[i].begin < segments[j].begin })

	return segments
}

func findMatchingSegments(line []byte, exprs exprsMap) []*segment {
	matches := make([]*segment, 0)

	for c, exs := range exprs {
		for _, expr := range exs {
			idxs := expr.FindAllSubmatchIndex(line, -1)

			if idxs == nil {
				continue
			}

			for _, idx := range idxs {
				matches = append(matches, newSegment(idx[0], idx[1]-1, c))
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].begin < matches[j].begin })

	return matches
}

func makeColorizedSegments(line []byte, matches []*segment) []*segment {
	segments := make([]*segment, 0, len(matches))

	segment := newSegment(0, -1, RESET)
	for i := 0; i < len(line); i++ {

		c := RESET
		overlaps := 0
		for _, m := range matches {
			if m.contains(i) {
				overlaps++

				if overlaps == 1 {
					c = m.c
				} else if overlaps > 1 {
					c = OVERLAP
					break
				}
			}
		}

		if segment.c == c {
			segment.end++
		} else {
			segments = append(segments, segment)
			segment = newSegment(i, i, c)
		}
	}

	return append(segments, segment)
}

func printColorized(line []byte, segments []*segment) {
	for _, s := range segments {
		fmt.Printf("%s%s%s", colors[s.c], string(line[s.begin:s.end+1]), colors[RESET])
	}
	fmt.Println()
}

func run(r io.Reader, exprs exprsMap) error {
	br := bufio.NewReader(r)

	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		if line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		matches := findMatchingSegments(line, exprs)
		colorized := makeColorizedSegments(line, matches)
		printColorized(line, colorized)
	}

	return nil
}

func parseArgs(args []string) (exprsMap, io.ReadCloser, error) {
	re := regexp.MustCompile(`^-(\w+)=(.*)`)
	exprs := make(exprsMap)
	var rc io.ReadCloser

	for i, arg := range args {
		parts := re.FindStringSubmatch(arg)

		if len(parts) != 3 {
			if i != len(args)-1 {
				return nil, nil, errors.New(fmt.Sprint("Invalid argument ", arg))
			}

			f, err := os.Open(arg)
			if err != nil {
				return nil, nil, err
			}

			rc = f
			break
		}

		log.Print("Parts: ", parts)

		col, pattern := color(parts[1]), parts[2]
		if colors[col] == "" {
			return nil, nil, errors.New(fmt.Sprint("Unrecognized color ", col))
		}

		if exprs[col] == nil {
			exprs[col] = make([]*regexp.Regexp, 0)
		}

		expr, err := regexp.Compile(pattern)
		if err != nil {
			return nil, nil, errors.New(fmt.Sprint("Invalid pattern ", pattern))
		}

		exprs[col] = append(exprs[col], expr)

	}

	if rc == nil {
		rc = os.Stdin
	}

	return exprs, rc, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <-<COLOOR>=<PATTERN> |-<COLOR>=<PATTERN> ...] [FILE]", os.Args[0])
	}

	exprs, file, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	run(file, exprs)
}
