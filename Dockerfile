FROM amd64/golang:1.18-buster

RUN apt-get update && apt-get install git

WORKDIR /usr/src/cudos-ondemand-minting-service

COPY . .

RUN go build -mod=readonly ./cmd/cudos-ondemand-minting-service

CMD ["/bin/bash", "-c", "./cudos-ondemand-minting-service"]