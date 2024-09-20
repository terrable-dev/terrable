package offline

import (
	_ "embed"
	"io"
	"os/exec"
	"sync"
)

type NodeProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
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

	stderr, err := cmd.StderrPipe()
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
		stdout: stdout,
		stderr: stderr,
		mutex:  sync.Mutex{},
	}, nil
}

func (np *NodeProcess) Execute(code string) error {
	np.mutex.Lock()
	defer np.mutex.Unlock()

	_, err := np.stdin.Write([]byte(code + "\n"))
	if err != nil {
		return err
	}

	return nil
}

func (np *NodeProcess) Close() {
	np.cmd.Process.Kill()
}
