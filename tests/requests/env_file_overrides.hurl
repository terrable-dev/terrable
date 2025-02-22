GET http://127.0.0.1:8081/echo-env-test
HTTP 200

[Asserts]
jsonpath "$.env.ENV_FILE_VAL" == "value-from-env-file"
jsonpath "$.env.ENV_FILE_OVERRIDE" == "overridden-value"
