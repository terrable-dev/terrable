//go:build e2e

package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const serverStartupTimeout = 60 * time.Second

var testServerInstance *testServer
var builtBinary *builtBinaryInfo

type testServer struct {
	baseURL string
	cmd     *exec.Cmd
	output  *safeBuffer
	waitCh  chan error
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

type builtBinaryInfo struct {
	path    string
	tempDir string
}

type readinessCheck struct {
	method         string
	path           string
	expectedStatus int
	body           string
	headers        map[string]string
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestMain(m *testing.M) {
	binary, err := buildTestBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build offline test binary: %v\n", err)
		os.Exit(1)
	}

	builtBinary = binary

	code := m.Run()

	if err := os.RemoveAll(binary.tempDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clean up offline test binary: %v\n", err)
		code = 1
	}

	os.Exit(code)
}

func TestOfflineCoreRequests(t *testing.T) {
	withTestServer(t, "samples/integration/core/offline.tf", "offline_core", "samples/integration/core/.env.sample", []readinessCheck{
		{method: http.MethodGet, path: "/", expectedStatus: http.StatusOK},
		{method: http.MethodPost, path: "/", expectedStatus: http.StatusOK},
		{method: http.MethodGet, path: "/echo-env-test", expectedStatus: http.StatusOK},
		{method: http.MethodGet, path: "/collision1", expectedStatus: http.StatusOK},
		{method: http.MethodPost, path: "/_sqs/SqsHandler", expectedStatus: http.StatusOK, body: "readiness"},
	}, func() {
		t.Run("echo GET request", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertHeader(t, "Content-Type", "application/json")
			response.assertJSONValue(t, "event.httpMethod", "GET")
		})

		t.Run("echo POST request", func(t *testing.T) {
			response := mustRequest(t, http.MethodPost, "/", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertJSONValue(t, "event.httpMethod", "POST")
		})

		t.Run("returns 404 for unsupported method", func(t *testing.T) {
			response := mustRequest(t, http.MethodDelete, "/", nil, nil)

			response.assertStatus(t, http.StatusNotFound)
			response.assertJSONValue(t, "message", "Not Found")
		})

		t.Run("returns 404 for missing route", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/missing-route-"+strconv.FormatInt(time.Now().UnixNano(), 10), nil, nil)

			response.assertStatus(t, http.StatusNotFound)
			response.assertJSONValue(t, "message", "Not Found")
		})

		t.Run("includes global environment variables", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertJSONValue(t, "env.GLOBAL_ENV", "global-env-var")
		})

		t.Run("applies env file overrides", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/echo-env-test", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertJSONValue(t, "env.ENV_FILE_VAL", "value-from-env-file")
			response.assertJSONValue(t, "env.ENV_FILE_OVERRIDE", "overridden-value")
		})

		t.Run("passes query string parameters", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/?firstQuery=123&secondQuery=hello", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertJSONValue(t, "queryStringParameters.firstQuery", "123")
			response.assertJSONValue(t, "queryStringParameters.secondQuery", "hello")
		})

		t.Run("supports callback handlers", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/echo-callback", nil, nil)
			response.assertStatus(t, http.StatusOK)
		})

		t.Run("sets standard response headers", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertHeader(t, "Content-Type", "application/json")
		})

		t.Run("returns delayed response and timing metadata", func(t *testing.T) {
			response := mustRequest(t, http.MethodGet, "/delayed", nil, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertJSONNumberAtLeast(t, "time", 150)

			if response.duration < 150*time.Millisecond {
				t.Fatalf("expected delayed request to take at least 150ms, took %s", response.duration)
			}
		})

		t.Run("builds an SQS-style event for queue handlers", func(t *testing.T) {
			response := mustRequest(t, http.MethodPost, "/_sqs/SqsHandler", nil, strings.NewReader("hello queue"))

			response.assertStatus(t, http.StatusOK)
			response.assertHeader(t, "Content-Type", "application/json")
			response.assertJSONNumberAtLeast(t, "recordCount", 1)
			response.assertJSONValue(t, "firstRecord.body", "hello queue")
			response.assertJSONValue(t, "firstRecord.eventSource", "aws:sqs")
			response.assertJSONValue(t, "firstRecord.eventSourceARN", "arn:aws:sqs:eu-west-1:000000000000:SqsHandler")
			response.assertJSONValue(t, "firstRecord.awsRegion", "eu-west-1")
			response.assertJSONValue(t, "firstRecord.approximateReceiveCount", "1")
		})

		t.Run("timeout request does not break later requests", func(t *testing.T) {
			timeoutResponse := mustRequest(t, http.MethodGet, "/timeout", nil, nil)
			timeoutResponse.assertStatus(t, http.StatusGatewayTimeout)

			followUpResponse := mustRequest(t, http.MethodGet, "/", nil, nil)
			followUpResponse.assertStatus(t, http.StatusOK)
		})

		t.Run("avoids handler collisions for same source file names", func(t *testing.T) {
			firstResponse := mustRequest(t, http.MethodGet, "/collision1", nil, nil)
			firstResponse.assertStatus(t, http.StatusOK)
			firstResponse.assertJSONValue(t, "collision", "1")

			secondResponse := mustRequest(t, http.MethodGet, "/collision2", nil, nil)
			secondResponse.assertStatus(t, http.StatusOK)
			secondResponse.assertJSONValue(t, "collision", "2")
		})
	})
}

func TestOfflineRESTAPICORSRequests(t *testing.T) {
	withTestServer(t, "samples/integration/rest-api-cors/offline.tf", "rest_api_cors", "", []readinessCheck{
		{method: http.MethodGet, path: "/", expectedStatus: http.StatusOK},
		{
			method:         http.MethodOptions,
			path:           "/echo-callback",
			expectedStatus: http.StatusNoContent,
			headers: map[string]string{
				"Origin":                        "https://app.example.com",
				"Access-Control-Request-Method": "GET",
			},
		},
	}, func() {
		t.Run("applies CORS response headers", func(t *testing.T) {
			headers := map[string]string{
				"Origin": "https://app.example.com",
			}

			response := mustRequest(t, http.MethodGet, "/", headers, nil)

			response.assertStatus(t, http.StatusOK)
			response.assertHeader(t, "Content-Type", "application/json")
			response.assertHeader(t, "Access-Control-Allow-Origin", "https://app.example.com")
			response.assertHeader(t, "Access-Control-Allow-Credentials", "true")
			response.assertHeader(t, "Access-Control-Expose-Headers", "x-terrable-request-id")
			response.assertHeader(t, "Vary", "Origin")
		})

		t.Run("applies implicit CORS OPTIONS headers on root", func(t *testing.T) {
			headers := map[string]string{
				"Origin":                         "https://app.example.com",
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "content-type,authorization",
			}

			response := mustRequest(t, http.MethodOptions, "/", headers, nil)

			response.assertStatus(t, http.StatusNoContent)
			response.assertHeader(t, "Access-Control-Allow-Origin", "https://app.example.com")
			response.assertHeader(t, "Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT")
			response.assertHeader(t, "Access-Control-Allow-Headers", "content-type, authorization")
			response.assertHeader(t, "Access-Control-Allow-Credentials", "true")
			response.assertHeader(t, "Access-Control-Max-Age", "600")
			response.assertHeader(t, "Vary", "Origin")
		})

		t.Run("applies implicit CORS OPTIONS headers on callback route", func(t *testing.T) {
			headers := map[string]string{
				"Origin":                        "https://app.example.com",
				"Access-Control-Request-Method": "GET",
			}

			response := mustRequest(t, http.MethodOptions, "/echo-callback", headers, nil)

			response.assertStatus(t, http.StatusNoContent)
			response.assertHeader(t, "Access-Control-Allow-Origin", "https://app.example.com")
			response.assertHeader(t, "Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT")
			response.assertHeader(t, "Access-Control-Allow-Headers", "content-type, authorization")
			response.assertHeader(t, "Access-Control-Allow-Credentials", "true")
			response.assertHeader(t, "Access-Control-Max-Age", "600")
			response.assertHeader(t, "Vary", "Origin")
		})
	})
}

func TestOfflineStartupReportsMissingHandlerSource(t *testing.T) {
	rootDir, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	output, err := runOfflineExpectFailure("samples/integration/invalid-handler/offline.tf", "invalid_handler", "")
	if err != nil {
		t.Fatalf("failed to run offline command: %v", err)
	}

	expectedResolvedPath := filepath.Join(rootDir, "samples", "integration", "invalid-handler", "src", "MissingHandler.ts")
	expectedFragments := []string{
		`Handler "MissingHandler" could not be loaded.`,
		`Configured source:`,
		`./src/MissingHandler.ts`,
		`Resolved path:`,
		expectedResolvedPath,
		`Problem:`,
		`no file exists at that path`,
		`Check the handler's "source" setting and try again.`,
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected offline output to contain %q, got:\n%s", fragment, output)
		}
	}

	if strings.Contains(output, "Starting terrable local server...") {
		t.Fatalf("expected offline startup to fail before the server banner, got:\n%s", output)
	}
}

func TestOfflineStartupReportsAllFailingHandlers(t *testing.T) {
	rootDir, err := repoRoot()
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	output, err := runOfflineExpectFailure("samples/integration/invalid-handlers/offline.tf", "invalid_handlers", "")
	if err != nil {
		t.Fatalf("failed to run offline command: %v", err)
	}

	expectedFragments := []string{
		"Terrable could not start because one or more handlers failed to prepare.",
		`Handler "MissingHandler" could not be loaded.`,
		`./src/MissingHandler.ts`,
		filepath.Join(rootDir, "samples", "integration", "invalid-handlers", "src", "MissingHandler.ts"),
		`Problem:`,
		`no file exists at that path`,
		`Handler "BrokenHandler" could not be compiled.`,
		`./src/BrokenHandler.ts`,
		filepath.Join(rootDir, "samples", "integration", "invalid-handlers", "src", "BrokenHandler.ts"),
		`Build errors:`,
		`Fix the handler code and try again.`,
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected offline output to contain %q, got:\n%s", fragment, output)
		}
	}

	if strings.Contains(output, "Starting terrable local server...") {
		t.Fatalf("expected offline startup to fail before the server banner, got:\n%s", output)
	}
}

type httpResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
	duration   time.Duration
}

func buildTestBinary() (*builtBinaryInfo, error) {
	rootDir, err := repoRoot()
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "terrable-e2e-*")
	if err != nil {
		return nil, err
	}

	binaryPath := filepath.Join(tempDir, "terrable"+exeSuffix())
	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCommand.Dir = rootDir
	buildOutput, err := buildCommand.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build failed: %w\n%s", err, string(buildOutput))
	}

	return &builtBinaryInfo{
		path:    binaryPath,
		tempDir: tempDir,
	}, nil
}

func withTestServer(t *testing.T, configPath, moduleName, envFilePath string, checks []readinessCheck, fn func()) {
	t.Helper()

	server, err := startTestServer(configPath, moduleName, envFilePath, checks)
	if err != nil {
		t.Fatalf("failed to start offline test server: %v", err)
	}

	testServerInstance = server
	defer func() {
		testServerInstance = nil
		if err := server.Stop(); err != nil {
			t.Fatalf("failed to stop offline test server cleanly: %v", err)
		}
	}()

	fn()
}

func startTestServer(configPath, moduleName, envFilePath string, checks []readinessCheck) (*testServer, error) {
	rootDir, err := repoRoot()
	if err != nil {
		return nil, err
	}

	port, err := reservePort()
	if err != nil {
		return nil, err
	}

	serverOutput := &safeBuffer{}
	args := []string{
		"offline",
		"-f", filepath.Join(rootDir, configPath),
		"-m", moduleName,
		"-p", strconv.Itoa(port),
		"--node-debug-port", "0",
	}

	if envFilePath != "" {
		args = append(args, "-envfile", filepath.Join(rootDir, envFilePath))
	}

	command := exec.Command(builtBinary.path, args...)
	command.Dir = rootDir

	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := command.Start(); err != nil {
		return nil, err
	}

	go io.Copy(serverOutput, stdout)
	go io.Copy(serverOutput, stderr)

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- command.Wait()
	}()

	server := &testServer{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		cmd:     command,
		output:  serverOutput,
		waitCh:  waitCh,
	}

	if err := waitForServer(server, checks, serverStartupTimeout); err != nil {
		_ = server.Stop()
		return nil, err
	}

	return server, nil
}

func runOfflineExpectFailure(configPath, moduleName, envFilePath string) (string, error) {
	rootDir, err := repoRoot()
	if err != nil {
		return "", err
	}

	port, err := reservePort()
	if err != nil {
		return "", err
	}

	args := []string{
		"offline",
		"-f", filepath.Join(rootDir, configPath),
		"-m", moduleName,
		"-p", strconv.Itoa(port),
		"--node-debug-port", "0",
	}

	if envFilePath != "" {
		args = append(args, "-envfile", filepath.Join(rootDir, envFilePath))
	}

	command := exec.Command(builtBinary.path, args...)
	command.Dir = rootDir

	output, waitErr := runCommandWithTimeout(command, 10*time.Second)
	if waitErr == nil {
		return output, fmt.Errorf("offline command unexpectedly succeeded:\n%s", output)
	}

	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return output, fmt.Errorf("offline command failed unexpectedly: %w\noutput:\n%s", waitErr, output)
	}

	return output, nil
}

func runCommandWithTimeout(command *exec.Cmd, timeout time.Duration) (string, error) {
	outputBuffer := &safeBuffer{}
	command.Stdout = outputBuffer
	command.Stderr = outputBuffer

	if err := command.Start(); err != nil {
		return "", err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- command.Wait()
	}()

	select {
	case err := <-waitCh:
		return outputBuffer.String(), err
	case <-time.After(timeout):
		_ = command.Process.Kill()
		<-waitCh
		return outputBuffer.String(), fmt.Errorf("command timed out after %s", timeout)
	}
}

func (s *testServer) Stop() error {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	if s.cmd.ProcessState == nil || !s.cmd.ProcessState.Exited() {
		_ = s.cmd.Process.Kill()
	}

	select {
	case err := <-s.waitCh:
		if err != nil && !errors.Is(err, os.ErrProcessDone) && !strings.Contains(err.Error(), "killed") {
			return fmt.Errorf("process exited unexpectedly: %w\nserver output:\n%s", err, s.output.String())
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out waiting for offline process to stop\nserver output:\n%s", s.output.String())
	}
}

func waitForServer(server *testServer, checks []readinessCheck, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case err := <-server.waitCh:
			return fmt.Errorf("offline process exited before becoming ready: %w\nserver output:\n%s", err, server.output.String())
		default:
		}

		ready, err := runReadinessChecks(client, server.baseURL, checks)
		if err == nil && ready {
			return nil
		}

		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("offline server did not become ready within %s\nserver output:\n%s", timeout, server.output.String())
}

func runReadinessChecks(client *http.Client, baseURL string, checks []readinessCheck) (bool, error) {
	for _, check := range checks {
		var body io.Reader
		if check.body != "" {
			body = strings.NewReader(check.body)
		}

		request, err := http.NewRequest(check.method, baseURL+check.path, body)
		if err != nil {
			return false, err
		}

		for key, value := range check.headers {
			request.Header.Set(key, value)
		}

		response, err := client.Do(request)
		if err != nil {
			return false, nil
		}
		response.Body.Close()

		if response.StatusCode != check.expectedStatus {
			return false, nil
		}
	}

	return true, nil
}

func mustRequest(t *testing.T, method, path string, headers map[string]string, body io.Reader) httpResponse {
	t.Helper()

	request, err := http.NewRequest(method, testServerInstance.baseURL+path, body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v\nserver output:\n%s", err, testServerInstance.output.String())
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return httpResponse{
		statusCode: response.StatusCode,
		headers:    response.Header.Clone(),
		body:       responseBody,
		duration:   time.Since(start),
	}
}

func (r httpResponse) assertStatus(t *testing.T, expected int) {
	t.Helper()

	if r.statusCode != expected {
		t.Fatalf("expected status %d, got %d. body=%s", expected, r.statusCode, string(r.body))
	}
}

func (r httpResponse) assertHeader(t *testing.T, name, expected string) {
	t.Helper()

	value := r.headers.Get(name)
	if value != expected {
		t.Fatalf("expected header %s=%q, got %q", name, expected, value)
	}
}

func (r httpResponse) assertJSONValue(t *testing.T, path, expected string) {
	t.Helper()

	value, err := r.jsonValue(path)
	if err != nil {
		t.Fatal(err)
	}

	stringValue, ok := value.(string)
	if !ok {
		t.Fatalf("expected %s to be a string, got %T", path, value)
	}

	if stringValue != expected {
		t.Fatalf("expected %s=%q, got %q", path, expected, stringValue)
	}
}

func (r httpResponse) assertJSONNumberAtLeast(t *testing.T, path string, minimum float64) {
	t.Helper()

	value, err := r.jsonValue(path)
	if err != nil {
		t.Fatal(err)
	}

	numberValue, ok := value.(float64)
	if !ok {
		t.Fatalf("expected %s to be a number, got %T", path, value)
	}

	if numberValue < minimum {
		t.Fatalf("expected %s >= %v, got %v", path, minimum, numberValue)
	}
}

func (r httpResponse) jsonValue(path string) (interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(r.body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JSON body %q: %w", string(r.body), err)
	}

	var current interface{} = payload
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path %s did not resolve to an object at %q", path, part)
		}

		next, ok := object[part]
		if !ok {
			return nil, fmt.Errorf("path %s missing key %q", path, part)
		}

		current = next
	}

	return current, nil
}

func reservePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func repoRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not determine current file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..")), nil
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}
