set -ex
GOOS=linux go build -o ./app .
# docker build -t rjlacr.azurecr.io/sample-client:latest .
# docker push rjlacr.azurecr.io/sample-client:latest
docker build -t mqiaoacr.azurecr.io/sample-client:latest .
docker push mqiaoacr.azurecr.io/sample-client:latest

