package kubectl

import "context"

// FakeResult describes a fake kubectl command result for tests.
type FakeResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// FakeCommand describes a fake kubectl command invocation for tests.
type FakeCommand struct {
	Args   []string
	Result FakeResult
}

// FakeRunner records kubectl commands and returns configured results.
type FakeRunner struct {
	results  []FakeResult
	commands []Command
}

// NewFakeRunner creates a fake kubectl runner for tests.
func NewFakeRunner(results ...FakeResult) *FakeRunner {
	return &FakeRunner{results: append([]FakeResult(nil), results...)}
}

// Run records a command and returns the next configured fake result.
func (runner *FakeRunner) Run(_ context.Context, command Command) (Result, error) {
	runner.commands = append(runner.commands, command)

	result := FakeResult{}
	if len(runner.results) > 0 {
		result = runner.results[0]
		runner.results = runner.results[1:]
	}

	converted := Result(result)
	if result.ExitCode != 0 {
		return converted, ExitError{Code: result.ExitCode, Stderr: result.Stderr}
	}
	return converted, nil
}

// Commands returns the recorded kubectl commands.
func (runner *FakeRunner) Commands() []Command {
	return append([]Command(nil), runner.commands...)
}
