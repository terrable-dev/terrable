# Ensures that two handlers configured with the same source file name 
# work as intended and do not have any collisions when transpiled

GET http://127.0.0.1:8081/collision1
HTTP 200
[Captures]
response_body: body
[Asserts]
jsonpath "$.collision" == "1"

GET http://127.0.0.1:8081/collision2
HTTP 200
[Captures]
response_body: body
[Asserts]
jsonpath "$.collision" == "2"
