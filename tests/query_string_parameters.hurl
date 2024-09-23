GET http://127.0.0.1:8080/?firstQuery=123&secondQuery=hello

HTTP 200
[Asserts]
jsonpath "$.queryString.firstQuery" == "123"
jsonpath "$.queryString.secondQuery" == "hello"
