package offline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func ServeHandler(handlerInstance *HandlerInstance, r *mux.Router) {
	inputFiles := handlerInstance.CompileHandler()
	go handlerInstance.WatchForChanges(inputFiles)

	np, err := NewNodeProcess()

	if err != nil {
		panic(err)
	}

	defer np.Close()

	go r.HandleFunc(handlerInstance.handlerConfig.Http.Path, func(w http.ResponseWriter, r *http.Request) {
		code := generateHandlerRuntimeCode(handlerInstance, r)

		err := np.Execute(code)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		var stdOutBuffer bytes.Buffer
		var stdErrBuffer bytes.Buffer

		done := make(chan bool)

		go processOutput(np.stdout, &stdOutBuffer, done)
		go processOutput(np.stderr, &stdErrBuffer, nil)

		<-done

		result, err := extractResult(stdOutBuffer.String())

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

	np.cmd.Wait()
}

func processOutput(r io.Reader, buffer *bytes.Buffer, done chan<- bool) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line)

		// Ignore terrable output marker
		if !strings.HasPrefix(line, "TERRABLE_RESULT_START:") {
			if strings.HasPrefix(line, "CODE_EXECUTION_COMPLETE") {
				// If complete statement, signal
				done <- true
				return
			} else {
				fmt.Println(line)
			}
		}
	}
}

func generateHandlerRuntimeCode(handler *HandlerInstance, r *http.Request) string {
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
	envVars, _ := json.Marshal(handler.envVars)

	return fmt.Sprintf(`
		var env = %s;

		for (const envKey in env) {
			process.env[envKey] = env[envKey];
		}

		delete require.cache[require.resolve('%s')];
		var transpiledFunction = require('%s');
		
	    var eventInput = %s;

		Promise
			.resolve(transpiledFunction.handler(eventInput))
			.then(result => {
				console.log("TERRABLE_RESULT_START:" + JSON.stringify(result) + ":TERRABLE_RESULT_END");
				complete();
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
				complete();
			})
	`, envVars, handler.GetExecutionPath(), handler.GetExecutionPath(), eventInputJSON)
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
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}
