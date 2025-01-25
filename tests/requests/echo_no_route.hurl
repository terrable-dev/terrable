# Hit a route that doesn't exist at all
GET http://127.0.0.1:8081/{{newUuid}}
HTTP 404

[Asserts]
jsonpath "$.message" == "Not Found"
