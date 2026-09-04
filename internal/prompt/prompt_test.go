package prompt

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestMultiSelectTogglesNumbersAndEnterConfirms(t *testing.T) {
	var out bytes.Buffer
	p := New(strings.NewReader("1 3\n2,3\n\n"), &out)
	got, err := p.MultiSelect("Pick", []string{"a", "b", "c"}, []bool{true, false, false})
	if err != nil {
		t.Fatal(err)
	}
	if want := []bool{false, true, false}; got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !strings.Contains(out.String(), "1. [x] a") || !strings.Contains(out.String(), "2. [ ] b") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestMultiSelectRejectsBadNumbersAndAsksAgain(t *testing.T) {
	var out bytes.Buffer
	p := New(strings.NewReader("9\nx\n2\n\n"), &out)
	got, err := p.MultiSelect("Pick", []string{"a", "b"}, []bool{false, false})
	if err != nil || got[0] || !got[1] {
		t.Fatalf("got %v, %v", got, err)
	}
	if strings.Count(out.String(), "is not a number between 1 and 2") != 2 {
		t.Fatalf("output = %q", out.String())
	}
}

func TestQuitAndEOFReturnErrQuit(t *testing.T) {
	for _, input := range []string{"q\n", "Q\n", ""} {
		p := New(strings.NewReader(input), &bytes.Buffer{})
		if _, err := p.MultiSelect("Pick", []string{"a"}, []bool{false}); !errors.Is(err, ErrQuit) {
			t.Fatalf("input %q: err = %v", input, err)
		}
		p = New(strings.NewReader(input), &bytes.Buffer{})
		if _, err := p.Confirm("Sure?", true); !errors.Is(err, ErrQuit) {
			t.Fatalf("confirm input %q: err = %v", input, err)
		}
	}
}

func TestConfirmReadsYesNoAndDefault(t *testing.T) {
	cases := []struct {
		input     string
		byDefault bool
		want      bool
	}{
		{"\n", true, true},
		{"\n", false, false},
		{"y\n", false, true},
		{"YES\n", false, true},
		{"n\n", true, false},
		{"maybe\nno\n", true, false},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		p := New(strings.NewReader(tc.input), &out)
		got, err := p.Confirm("Sure?", tc.byDefault)
		if err != nil || got != tc.want {
			t.Fatalf("input %q default %t: got %t, %v", tc.input, tc.byDefault, got, err)
		}
		if tc.byDefault && !strings.Contains(out.String(), "[Y/n]") || !tc.byDefault && !strings.Contains(out.String(), "[y/N]") {
			t.Fatalf("hint missing in %q", out.String())
		}
	}
}

func TestChooseTakesOneNumberAndRejectsTheRest(t *testing.T) {
	var out bytes.Buffer
	p := New(strings.NewReader("\n4\nx\n2\n"), &out)
	got, err := p.Choose("Which?", []string{"a", "b", "c"})
	if err != nil || got != 1 {
		t.Fatalf("got %d, %v", got, err)
	}
	if !strings.Contains(out.String(), "  2. b") || strings.Count(out.String(), "is not a number between 1 and 3") != 3 {
		t.Fatalf("output = %q", out.String())
	}
	if _, err := New(strings.NewReader("q\n"), &out).Choose("Which?", []string{"a"}); !errors.Is(err, ErrQuit) {
		t.Fatalf("q: err = %v", err)
	}
}

func TestAskReturnsTheLineAndEnterAlone(t *testing.T) {
	var out bytes.Buffer
	p := New(strings.NewReader("  me/work-skills \n\n"), &out)
	got, err := p.Ask("Repo?")
	if err != nil || got != "me/work-skills" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := p.Ask("Repo?"); err != nil || got != "" {
		t.Fatalf("enter alone: got %q, %v", got, err)
	}
	if !strings.HasPrefix(out.String(), "Repo? ") {
		t.Fatalf("output = %q", out.String())
	}
}
