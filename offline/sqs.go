package offline

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Admiral-Piett/goaws/app/gosqs"
	"github.com/Admiral-Piett/goaws/app/router"
)

func startSqsServer() {
	var wg sync.WaitGroup
	defer wg.Wait()

	quit := make(chan bool)
	r := router.New()

	go gosqs.PeriodicTasks(1*time.Second, quit)
	wg.Add(1)

	go func() {
		go http.ListenAndServe("127.0.0.1:4100", r)
	}()
}

func addSqsQueue(queueName string) error {
	baseURL := "http://127.0.0.1:4100"

	// Prepare form data
	data := url.Values{}
	data.Set("Action", "CreateQueue")
	data.Set("QueueName", queueName)

	// Create the request
	req, err := http.NewRequest("POST", baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

type ReceiveMessageResponse struct {
	Messages []struct {
		Body string `xml:"Body"`
	} `xml:"ReceiveMessageResult>Message"`
}

func getSqsMessages(queueName string) ([]string, error) {
	baseURL := "http://127.0.0.1:4100"

	// Prepare form data
	data := url.Values{}
	data.Set("Action", "ReceiveMessage")
	data.Set("QueueUrl", fmt.Sprintf("%s/queue/%s", baseURL, url.QueryEscape(queueName)))
	data.Set("MaxNumberOfMessages", "10")

	// Create the request
	req, err := http.NewRequest("POST", baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result ReceiveMessageResponse
	err = xml.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing XML: %v", err)
	}

	messages := make([]string, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = msg.Body
	}

	return messages, nil
}
