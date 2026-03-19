GET http://127.0.0.1:8081/
Origin: https://app.example.com
HTTP 200

[Asserts]
header "Content-Type" == "application/json"
header "Access-Control-Allow-Origin" == "https://app.example.com"
header "Access-Control-Allow-Credentials" == "true"
header "Access-Control-Expose-Headers" == "x-terrable-request-id"
header "Vary" == "Origin"
