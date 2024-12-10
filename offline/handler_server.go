package offline

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/mux"
)

var handlerExecutionMutex sync.Mutex

func ServeHandler(handlerInstance *HandlerInstance, r *mux.Router) {
	inputFiles := handlerInstance.CompileHandler()
	go handlerInstance.WatchForChanges(inputFiles)

	np, err := GetNodeProcess()

	if err != nil {
		panic(err)
	}

	defer np.Close()

	for method, path := range handlerInstance.handlerConfig.Http {
		// TODO: Emulate API Gateway's 404 for missing routes / methods

		go r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			handlerExecutionMutex.Lock()
			defer handlerExecutionMutex.Unlock()

			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()

			code := generateHandlerRuntimeCode(handlerInstance, r)

			np.Execute(code)

			parsedOutputChan := make(chan *struct {
				handlerResult *handlerResult
				err           error
			}, 1)

			fmt.Printf("%s %s (%s) \n", r.Method, r.URL.Path, handlerInstance.handlerConfig.Name)
			start := time.Now()

			// stdout processing
			go func() {
				scanner := bufio.NewReader(np.stdout)

				for {
					select {
					case <-ctx.Done():
						return
					default:
						line, _ := scanner.ReadString('\n')

						if strings.HasPrefix(line, "TERRABLE_RESULT_START") {
							extractedResult, err := extractResult(line)

							parsedOutputChan <- &struct {
								handlerResult *handlerResult
								err           error
							}{
								handlerResult: extractedResult,
								err:           err,
							}

							return
						}

						if strings.HasPrefix(line, "CODE_EXECUTION_COMPLETE") {
							continue
						}

						fmt.Println(line)
					}
				}
			}()

			// stderr processing
			go func() {
				scanner := bufio.NewReader(np.stderr)
				errorColour := color.New(color.FgHiRed).SprintFunc()

				for {
					select {
					case <-ctx.Done():
						return
					default:
						line, _ := scanner.ReadString('\n')
						fmt.Println(errorColour(line))
					}
				}
			}()

			select {
			case parsed := <-parsedOutputChan:
				if parsed.err != nil {
					fmt.Println(err)
					w.WriteHeader(500)
					w.Write([]byte{})
					return
				}

				// Set response headers
				for k, header := range parsed.handlerResult.Headers {
					w.Header().Set(k, header)
				}

				// Write status code
				w.WriteHeader(int(parsed.handlerResult.StatusCode))

				// Write the body
				w.Write([]byte(parsed.handlerResult.Body))
				fmt.Printf("Completed in %.dms\n\n", time.Since(start).Milliseconds())
			case <-ctx.Done():
				// Handle timeout
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte{})

				fmt.Printf("Request timed out\n")
				fmt.Printf("Completed in %.dms\n\n", time.Since(start).Milliseconds())
				return
			}
		}).Methods(method)
	}

	np.cmd.Wait()
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

	// Set body
	if len(body) > 0 {
		bodyValue = string(body)
	} else {
		bodyValue = nil
	}

	if bodyValue == "" {
		bodyValue = nil
	}

	// Set query string params
	var queryParamsValue interface{}

	if len(queryParams) > 0 {
		queryParamsValue = queryParams
	} else {
		queryParamsValue = nil
	}

	// Set path parameters
	pathParams := mux.Vars(r)

	if len(pathParams) < 1 {
		pathParams = nil
	}

	eventInput := map[string]interface{}{
		"body":                  bodyValue,
		"queryStringParameters": queryParamsValue,
		"httpMethod":            r.Method,
		"path":                  r.URL.Path,
		"headers":               headers,
		"pathParameters":        pathParams,
	}

	eventInputJSON, _ := json.Marshal(eventInput)

	// Create a merge of handler-defined env vars
	// and any OS env vars to be passed into the function handler
	envVars := make(map[string]string)
	processEnvVars := os.Environ()

	for _, env := range processEnvVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			envVars[key] = value
		}
	}

	for key, value := range handler.envVars {
		envVars[key] = value
	}

	mergedEnvVars, _ := json.Marshal(envVars)

	return fmt.Sprintf(`
		const env = %s;
		process.env = {};

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
	`, mergedEnvVars, handler.GetExecutionPath(), handler.GetExecutionPath(), eventInputJSON)
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
