package offline

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type NodeProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mutex  sync.Mutex
}

//go:embed node_handler_wrapper.js
var NODE_HANDLER_WRAPPER string

func NewNodeProcess() (*NodeProcess, error) {
	cmd := exec.Command("node", "--inspect=9229", "-e", NODE_HANDLER_WRAPPER)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Env = []string{}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return &NodeProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		mutex:  sync.Mutex{},
	}, nil
}

func (np *NodeProcess) Execute(code string) (string, error) {
	np.mutex.Lock()
	defer np.mutex.Unlock()

	_, err := np.stdin.Write([]byte(code + "\n"))
	if err != nil {
		return "", err
	}

	var output string

	for {
		line, err := np.stdout.ReadString('\n')
		if err != nil {
			return "", err
		}

		output += line
		if strings.TrimSpace(line) == "CODE_EXECUTION_COMPLETE" {
			break
		}
	}

	return output, nil
}

func (np *NodeProcess) Close() {
	fmt.Println("Defer np close")
	np.cmd.Process.Kill()
}
