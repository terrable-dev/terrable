GET http://127.0.0.1:8081/
HTTP 200

[Asserts]
jsonpath "$.env.GLOBAL_ENV" == "global-env-var"
