set -ex
GOOS=linux go build -o ./app .
docker build -t rjlacr.azurecr.io/sample-client:latest .
docker push rjlacr.azurecr.io/sample-client:latest

