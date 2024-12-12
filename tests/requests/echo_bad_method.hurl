# Hit a route that exists but ot for the method in the request
DELETE http://127.0.0.1:8081/
HTTP 404

[Asserts]
jsonpath "$.message" == "Not Found"

# Hit a route that doesn't exist at all
GET http://127.0.0.1:8081/{{uuid}}
HTTP 404

[Asserts]
jsonpath "$.message" == "Not Found"
