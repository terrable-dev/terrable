GET http://127.0.0.1:8081/
HTTP 200

[Asserts]
jsonpath "$.env.GLOBAL_ENV" == "global-env-var"
jsonpath "$.env.ECHO_ENV" == "echo-env"

GET http://127.0.0.1:8081/echo-no-env
HTTP 200

[Asserts]
jsonpath "$.env.GLOBAL_ENV" == "global-env-var"
jsonpath "$.env.ECHO_ENV" not exists

