go build ../
./terrable offline -f "../samples/simple/simple-api.tf" -m "simple_api" -p "8081" -envfile "../samples/simple/.env.sample"