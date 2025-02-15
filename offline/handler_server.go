package offline

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var handlerExecutionMutex sync.Mutex

type HandlerOutput struct {
	handlerResult *handlerResult
	err           error
}

func ServeHandler(handlerInstance *HandlerInstance, r *mux.Router) {
	inputFiles := handlerInstance.CompileHandler()
	go handlerInstance.WatchForChanges(inputFiles)

	np, err := GetNodeProcess()

	if err != nil {
		panic(err)
	}

	defer np.Close()

	handleRequestFunc := func(w http.ResponseWriter, r *http.Request, code string) {
		handlerExecutionMutex.Lock()
		defer handlerExecutionMutex.Unlock()

		np.Execute(code)

		outputChannel := make(chan HandlerOutput, 1)
		errorChannel := make(chan error, 1)

		fmt.Printf("%s %s (%s) \n", r.Method, r.URL.Path, handlerInstance.handlerConfig.Name)
		start := time.Now()

		go func() {
			if err := processOutputStream(np, outputChannel); err != nil {
				errorChannel <- err
			}
		}()

		// Start error processing with done channel
		go processErrorStream(np)
		sendResult(start, w, outputChannel)
	}

	for method, path := range handlerInstance.handlerConfig.Http {
		r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			code := generateHttpHandlerRuntimeCode(handlerInstance, r)
			handleRequestFunc(w, r, code)
		}).Methods(method)
	}

	for range handlerInstance.handlerConfig.Sqs {
		r.HandleFunc(fmt.Sprintf("/_sqs/%s", handlerInstance.handlerConfig.Name), func(w http.ResponseWriter, r *http.Request) {
			code := generateSqsHandlerRuntimeCode(handlerInstance, r)
			handleRequestFunc(w, r, code)
		}).Methods("POST")
	}

	np.cmd.Wait()
}

func sendResult(startTime time.Time, w http.ResponseWriter, outputChannel chan HandlerOutput) {
	parsed := <-outputChannel
	if parsed.err != nil {
		fmt.Println(parsed.err)
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
	fmt.Printf("Completed in %.dms\n\n", time.Since(startTime).Milliseconds())
}

func processOutputStream(np *NodeProcess, resultChan chan<- HandlerOutput) error {
	scanner := bufio.NewReader(np.stdout)

	for {
		line, err := scanner.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if strings.HasPrefix(line, "TERRABLE_RESULT_START") {
			extractedResult, err := extractResult(line)
			resultChan <- HandlerOutput{
				handlerResult: extractedResult,
				err:           err,
			}
			return nil
		}

		if strings.HasPrefix(line, "CODE_EXECUTION_COMPLETE") {
			continue
		}

		fmt.Println(line)
	}
}

func processErrorStream(np *NodeProcess) {
	scanner := bufio.NewReader(np.stderr)
	errorColour := color.New(color.FgHiRed).SprintFunc()

	line, _ := scanner.ReadString('\n')
	fmt.Println(errorColour(line))
}

func generateHttpHandlerRuntimeCode(handler *HandlerInstance, r *http.Request) string {
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
	envVars := generateEnvVars(handler)
	return generateJSCode(string(envVars), handler.GetExecutionPath(), string(eventInputJSON), handler.handlerConfig.Timeout)
}

func generateEnvVars(handler *HandlerInstance) string {
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
	return string(mergedEnvVars)
}

func generateSqsHandlerRuntimeCode(handler *HandlerInstance, r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Create an SQS message
	message := map[string]interface{}{
		"messageId": uuid.New().String(),
		"body":      string(body),
		"attributes": map[string]interface{}{
			"ApproximateReceiveCount":          "1",
			"SentTimestamp":                    fmt.Sprintf("%d", time.Now().UnixNano()/1e6),
			"SenderId":                         "SIMULATOR",
			"ApproximateFirstReceiveTimestamp": fmt.Sprintf("%d", time.Now().UnixNano()/1e6),
		},
		"messageAttributes": map[string]interface{}{},
		"md5OfBody":         fmt.Sprintf("%x", md5.Sum(body)),
		"eventSource":       "aws:sqs",
		"eventSourceARN":    fmt.Sprintf("arn:aws:sqs:eu-west-1:000000000000:%s", handler.handlerConfig.Name),
		"awsRegion":         "eu-west-1",
	}

	// Create the SQS event structure
	eventInput := map[string]interface{}{
		"Records": []interface{}{message},
	}

	eventInputJSON, _ := json.Marshal(eventInput)
	envVars := generateEnvVars(handler)
	return generateJSCode(string(envVars), handler.GetExecutionPath(), string(eventInputJSON), handler.handlerConfig.Timeout)
}

func generateJSCode(envVars, executionPath, eventInputJSON string, timeoutSeconds int) string {
	return fmt.Sprintf(`
        const env = %s;
        process.env = {};

        for (const envKey in env) {
            process.env[envKey] = env[envKey];
        }

        delete require.cache[require.resolve('%s')];
        var transpiledFunction = require('%s');
        
        var eventInput = %s;
        const endTime = Date.now() + (%d * 1000);

        // Create a fake context object
        const context = {
			functionName: "local-function",
			functionVersion: "\$LATEST",
			invokedFunctionArn: "local:lambda",
			memoryLimitInMB: "128",
			awsRequestId: "local-" + Date.now(),
			logGroupName: "local-group",
			logStreamName: "local-stream",
			getRemainingTimeInMillis: () => {
                const remaining = endTime - Date.now();
                return remaining > 0 ? remaining : 0;
            },
			callbackWaitsForEmptyEventLoop: true
    	};

        // Create a timeout promise
        const timeoutPromise = new Promise((resolve) => {
            setTimeout(() => {
				resolve({ statusCode: 504 })
            }, %d * 1000);
        });

		// Main execution promise
        const executionPromise = new Promise((resolve, reject) => {
            const callback = (error, result) => {
                if (error) {
                    reject(error);
                } else {
                    resolve(result);
                }
            };

            const handlerResult = transpiledFunction.handler(eventInput, context, callback);

            if (handlerResult && typeof handlerResult.then === 'function') {
                handlerResult.then(resolve).catch(reject);
            } else if (!handlerResult) {
                // If handlerResult is undefined, it means the function is using callbacks
            } else {
                resolve(handlerResult);
            }
        });

        // Race between execution and timeout
        Promise.race([executionPromise, timeoutPromise])
        .then(result => {
			console.log("TERRABLE_RESULT_START:" + JSON.stringify({ statusCode: 200, ...result }) + ":TERRABLE_RESULT_END");
        })
        .catch(error => {
            console.error(error);
            console.log("TERRABLE_RESULT_START:" + JSON.stringify({
                statusCode: error.message.includes('timed out') ? 408 : 500,
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    message: error.message.includes('timed out') ? "Function timed out" : "Internal server error",
                    errorMessage: error.message,
                    errorType: error.name,
                    stackTrace: error.stack
                })
            }) + ":TERRABLE_RESULT_END");
        })
        .finally(() => {
            complete();
        });
    `, envVars, executionPath, executionPath, eventInputJSON, timeoutSeconds, timeoutSeconds)
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
