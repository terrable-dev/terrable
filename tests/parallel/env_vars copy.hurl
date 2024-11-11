GET http://127.0.0.1:8081/collision1
HTTP 200

[Asserts]
jsonpath "$.collision" == "1"

GET http://127.0.0.1:8081/collision2
HTTP 200

[Asserts]
jsonpath "$.collision" == "2"
