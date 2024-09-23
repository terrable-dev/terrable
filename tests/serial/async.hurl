GET http://127.0.0.1:8080/async

HTTP 200
[Asserts]
jsonpath "$.time" >= 150
duration >= 150