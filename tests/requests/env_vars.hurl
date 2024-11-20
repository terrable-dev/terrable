# /echo-no-vars should have only the global env vars

GET http://127.0.0.1:8081/echo-no-vars
HTTP 200

[Asserts]
jsonpath "$.env.GLOBAL_VARIABLE" == "global-variable"
jsonpath "$.env.ECHO_VARIABLE" not exists

# / should have global and local env vars

GET http://127.0.0.1:8081/
HTTP 200

[Asserts]
jsonpath "$.env.GLOBAL_VARIABLE" == "global-variable"
jsonpath "$.env.ECHO_VARIABLE" == "echo-variable"
