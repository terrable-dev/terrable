# Tests that the SQS Handler endpoint is exposed and executes

POST http://127.0.0.1:8081/_sqs/SqsHandler
HTTP 200

[Asserts]
duration >= 300
