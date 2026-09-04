// Package prompt asks the human questions on a plain terminal: a numbered
// multi-select, a numbered single pick, a yes or no, and a line of text. Line
// based, no TUI library.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ErrQuit is returned when the human types q instead of answering.
var ErrQuit = errors.New("quit")

// Prompt reads answers from in and writes questions to out.
type Prompt struct {
	reader *bufio.Reader
	out    io.Writer
}

// New wraps the terminal streams.
func New(in io.Reader, out io.Writer) *Prompt {
	return &Prompt{reader: bufio.NewReader(in), out: out}
}

// MultiSelect shows labels as a numbered list with selected marking the
// preselected ones. Typing numbers toggles them, Enter keeps the current
// selection, q quits. Returns the final selection.
func (p *Prompt) MultiSelect(title string, labels []string, selected []bool) ([]bool, error) {
	chosen := append([]bool(nil), selected...)
	for {
		fmt.Fprintf(p.out, "\n%s\n", title)
		for index, label := range labels {
			mark := " "
			if chosen[index] {
				mark = "x"
			}
			fmt.Fprintf(p.out, "  %d. [%s] %s\n", index+1, mark, label)
		}
		fmt.Fprint(p.out, "Numbers toggle (for example 1 3), Enter continues, q quits: ")
		answer, err := p.readLine()
		if err != nil {
			return nil, err
		}
		if answer == "" {
			return chosen, nil
		}
		valid := true
		for _, field := range strings.Fields(strings.ReplaceAll(answer, ",", " ")) {
			number, err := strconv.Atoi(field)
			if err != nil || number < 1 || number > len(labels) {
				fmt.Fprintf(p.out, "%q is not a number between 1 and %d.\n", field, len(labels))
				valid = false
				break
			}
			chosen[number-1] = !chosen[number-1]
		}
		if !valid {
			continue
		}
	}
}

// Confirm asks a yes or no question. Enter picks the default.
func (p *Prompt) Confirm(question string, byDefault bool) (bool, error) {
	hint := "[y/N]"
	if byDefault {
		hint = "[Y/n]"
	}
	for {
		fmt.Fprintf(p.out, "%s %s ", question, hint)
		answer, err := p.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return byDefault, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		fmt.Fprintln(p.out, "Answer y or n.")
	}
}

// Choose shows labels as a numbered list and returns the index of the one
// picked. There is no default: Enter alone asks again, q quits.
func (p *Prompt) Choose(title string, labels []string) (int, error) {
	for {
		fmt.Fprintf(p.out, "\n%s\n", title)
		for index, label := range labels {
			fmt.Fprintf(p.out, "  %d. %s\n", index+1, label)
		}
		fmt.Fprintf(p.out, "Pick 1 to %d, q quits: ", len(labels))
		answer, err := p.readLine()
		if err != nil {
			return 0, err
		}
		number, err := strconv.Atoi(answer)
		if err == nil && number >= 1 && number <= len(labels) {
			return number - 1, nil
		}
		fmt.Fprintf(p.out, "%q is not a number between 1 and %d.\n", answer, len(labels))
	}
}

// Ask asks for a line of text. Enter alone answers "".
func (p *Prompt) Ask(question string) (string, error) {
	fmt.Fprintf(p.out, "%s ", question)
	return p.readLine()
}

func (p *Prompt) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("[JSTACK-PROMPT-READ] cannot read the answer: %w; rerun with --yes and --harness to skip the prompts", err)
	}
	line = strings.TrimSpace(line)
	if strings.EqualFold(line, "q") || (errors.Is(err, io.EOF) && line == "") {
		return "", ErrQuit
	}
	return line, nil
}
