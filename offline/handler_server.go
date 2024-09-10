package offline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
)

func ServeHandler(handlerInstance *HandlerInstance, r *mux.Router) {
	inputFiles := handlerInstance.CompileHandler()
	go handlerInstance.WatchForChanges(inputFiles)

	go r.HandleFunc(handlerInstance.handlerConfig.Http.Path, func(w http.ResponseWriter, r *http.Request) {
		code := wrapHandlerCode(handlerInstance.handlerCode, r)

		output := executeNodeCode(code, handlerInstance.envVars)
		result, err := extractResult(output)

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			w.Write([]byte{})
			return
		}

		// Set response headers
		for k, header := range result.Headers {
			w.Header().Set(k, header)
		}

		// Write status code
		w.WriteHeader(int(result.StatusCode))

		// Write the body
		w.Write([]byte(result.Body))
	})
}

func wrapHandlerCode(handlerCode string, r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	queryParams := make(map[string]string)

	for key, values := range r.URL.Query() {
		queryParams[key] = values[len(values)-1] // Take the last value
	}

	headers := make(map[string]string)

	for key, values := range r.Header {
		headers[key] = values[0]
	}

	// Format for API Gateway behaviours
	var bodyValue interface{}

	if len(body) > 0 {
		bodyValue = string(body)
	} else {
		bodyValue = nil
	}

	if bodyValue == "" {
		bodyValue = nil
	}

	var queryParamsValue interface{}

	if len(queryParams) > 0 {
		queryParamsValue = queryParams
	} else {
		queryParamsValue = nil
	}

	vars := mux.Vars(r)

	if len(vars) < 1 {
		vars = nil
	}

	eventInput := map[string]interface{}{
		"body":                  bodyValue,
		"queryStringParameters": queryParamsValue,
		"httpMethod":            r.Method,
		"path":                  r.URL.Path,
		"headers":               headers,
		"pathParameters":        vars,
	}

	eventInputJSON, _ := json.Marshal(eventInput)

	return fmt.Sprintf(`
        var eventInput = %s;
	
		%s
	
		Promise
			.resolve(exports.handler(eventInput))
			.then(result => {
				console.log("TERRABLE_RESULT_START:" + JSON.stringify(result) + ":TERRABLE_RESULT_END");
			})
			.catch(error => {
				console.error(error);
				console.log("TERRABLE_RESULT_START:" + JSON.stringify({
					statusCode: 500,
					headers: {
						"Content-Type": "application/json",
					},
					body: JSON.stringify({
						message: "Internal server error",
						errorMessage: error.message,
						errorType: error.name,
						stackTrace: error.stack
					})
				}) + ":TERRABLE_RESULT_END")
			})
	`, eventInputJSON, handlerCode)
}

func executeNodeCode(code string, environmentVariables map[string]interface{}) string {
	cmd := exec.Command("node", "-e", string(code))

	cmd.Env = []string{}

	for key, value := range environmentVariables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", key, value))
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	cmd.Start()

	var resultBuffer bytes.Buffer
	var logBuffer bytes.Buffer

	go processOutput(stdout, &logBuffer)
	go processOutput(stderr, &resultBuffer)

	cmd.Wait()

	return logBuffer.String()
}

func processOutput(r io.Reader, buffer *bytes.Buffer) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// Ignore terrable output marker
		if !strings.HasPrefix(line, "TERRABLE_RESULT_START:") {
			fmt.Println(line)
		}

		buffer.WriteString(line)
	}
}

func extractResult(output string) (*handlerResult, error) {
	startIndex := strings.Index(output, "TERRABLE_RESULT_START:") + len("TERRABLE_RESULT_START:")
	endIndex := strings.Index(output, ":TERRABLE_RESULT_END")

	var result string

	if startIndex >= 0 && endIndex >= 0 && endIndex > startIndex {
		result = output[startIndex:endIndex]
	} else {
		return nil, fmt.Errorf("no TERRABLE_RESULT markers found. Unable to parse result")
	}

	// Parse the JSON result
	var handlerResult handlerResult

	if err := json.Unmarshal([]byte(result), &handlerResult); err != nil {
		return nil, err
	}

	// Extract statusCode and body
	return &handlerResult, nil
}

type handlerResult struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:headers`
	Body       string            `json:body`
}
