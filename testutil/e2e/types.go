package main

import "os/exec"

type TestResult struct {
	eventID string `json:"eventID"`
	found   bool   `json:"eventID"`
	passed  bool   `json:"passed"`
	line    string `json:"comment"`
	err     error  `json:"err"`
	parent  string `json:"parent"`
	failNow bool   `json:"failNow"`
}
type LogLine struct {
	line   string `json:"line"`
	parent string `json:"parent"`
}

type CMD struct {
	stateID      string                   `json:"stateID"`
	homepath     string                   `json:"homepath"`
	cmd          string                   `json:"cmd"`
	filter       []string                 `json:"filter"`
	testing      bool                     `json:"testing"`
	test         TestProc                 `json:"test"`
	results      map[string][]*TestResult `json:"results"`
	dep          *State                   `json:"dep"`
	failed       *bool                    `json:"failed"`
	requireAlive bool                     `json:"requireAlive"`
	debug        *bool                    `json:"debug"`
	stdout       *bool                    `json:"debug"`
}

type State struct {
	id           string                   `json:"id"`
	finished     *bool                    `json:"finished"`
	awaiting     map[string]Await         `json:"awating"`
	testing      bool                     `json:"testing"`
	test         TestProc                 `json:"test"`
	results      map[string][]*TestResult `json:"results"`
	depending    []*State                 `json:"depending"`
	cmd          *exec.Cmd                `json:"cmd"`
	failed       *bool                    `json:"failed"`
	requireAlive bool                     `json:"requireAlive"`
	lastLine     *string                  `json:"lastLine"`
	debug        *bool                    `json:"debug"`
	stdout       *bool                    `json:"debug"`
}

type Await struct {
	done *bool                   `json:"done"`
	pass *bool                   `json:"pass"`
	f    func(string) TestResult `json:"f"`
	msg  string                  `json:"msg"`
}

type TestProc struct {
	expectedEvents   []string                              `json:"expectedEvents"`
	unexpectedEvents []string                              `json:"unexpectedEvents"`
	tests            map[string](func(LogLine) TestResult) `json:"tests"`
	strict           bool                                  `json:"strict"`
	filter           []string                              `json:"filter"`
}

type Common struct {
	expectedEvents   []string                              `json:"expectedEvents"`
	unexpectedEvents []string                              `json:"unexpectedEvents"`
	tests            map[string](func(LogLine) TestResult) `json:"tests"`
	strict           bool                                  `json:"strict"`
	filter           []string                              `json:"filter"`
}
