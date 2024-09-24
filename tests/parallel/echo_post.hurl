POST http://127.0.0.1:8080/
HTTP 200

[Asserts]
jsonpath "$.event.httpMethod" == "POST"
