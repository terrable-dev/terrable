# Tests that the SQS Handler endpoint is exposed and executes

GET http://127.0.0.1:8081/timeout
HTTP 504

GET http://127.0.0.1:8081/
HTTP 200
