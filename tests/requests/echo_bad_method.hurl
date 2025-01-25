# Hit a route that exists but not for the method in the request
DELETE http://127.0.0.1:8081/
HTTP 404

[Asserts]
jsonpath "$.message" == "Not Found"
