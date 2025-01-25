# Some routes in the terrable configuration are deliberately missing trailing prefixes
# These tests check that the routes are normalised correctly

PUT http://127.0.0.1:8081/
HTTP 200

PUT http://127.0.0.1:8081/echo-callback
HTTP 200