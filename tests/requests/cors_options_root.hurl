OPTIONS http://127.0.0.1:8081/
Origin: https://app.example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: content-type,authorization
HTTP 204

[Asserts]
header "Access-Control-Allow-Origin" == "https://app.example.com"
header "Access-Control-Allow-Methods" == "GET, OPTIONS, POST, PUT"
header "Access-Control-Allow-Headers" == "content-type, authorization"
header "Access-Control-Allow-Credentials" == "true"
header "Access-Control-Max-Age" == "600"
header "Vary" == "Origin"
