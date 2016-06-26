package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

func (c CommandResult) String() string {
	s, _ := json.Marshal(c)
	return string(s)
}

// CommandResult - XXX
type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Command  string `json:"command"`
}

// Checks - XXX
type Checks struct {
	Config Config
}

// Description - XXX
func (c *Checks) Description() string {
	return "Collects data from Sensu plugins"
}

var sampleConfig = `
#   Available config options:
#
#    [
#        "metrics-es-node-graphite.rb",
#        "metrics-net.rb",
#        "metrics-redis-graphite.rb",
#        "metrics-iostat-extended.rb"
#    ]
#
#    List of preinstalled sensu plugins + params
#
# Config location: /etc/opt/amonagent/plugins-enabled/checks.conf
`

// SampleConfig - XXX
func (c *Checks) SampleConfig() string {
	return sampleConfig
}

// Config - XXX
type Config struct {
	Commands []string `mapstructure:"commands"`
}

// SetConfigDefaults - XXX
func (c *Checks) SetConfigDefaults(configPath string) error {
	jsonFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Can't read config file: %s %v\n", configPath, err)
	}
	var Commands []string
	if err := json.Unmarshal(jsonFile, &Commands); err != nil {
		fmt.Printf("Can't decode JSON file: %v\n", err)
	}

	c.Config.Commands = Commands

	return nil
}

// ExecWithExitCode - XXX
// Source: http://stackoverflow.com/questions/10385551/get-exit-code-go
func ExecWithExitCode(command string) (CommandResult, error) {
	parts := strings.Fields(command)
	head := parts[0]
	parts = parts[1:]
	cmd := exec.Command(head, parts...)
	output := CommandResult{Command: command}

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Start(); err != nil {
		return output, err
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				output.ExitCode = status.ExitStatus()
			}
		} else {
			return output, err
		}
	}
	output.Output = out.String()

	return output, nil

}

// Collect - XXX
func (c *Checks) Collect(configPath string) (interface{}, error) {
	c.SetConfigDefaults(configPath)
	var wg sync.WaitGroup
	var result []CommandResult

	for _, v := range c.Config.Commands {
		wg.Add(1)

		go func(command string) {

			CheckResult, err := ExecWithExitCode(command)
			if err != nil {
				fmt.Println("Can't execute command: ", err)
			}

			result = append(result, CheckResult)
			defer wg.Done()
		}(v)

	}
	wg.Wait()

	return result, nil
}

func main() {
	c := Checks{}

	c.Collect("/etc/amonagent/plugins-enabled/checks.conf")
}
