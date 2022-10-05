FROM amd64/golang:1.18-buster

RUN apt-get update && apt-get install git
RUN apt install -y jq build-essential

WORKDIR /usr/src/cudos-ondemand-minting-service

COPY . .

CMD ["/bin/bash", "-c", "./integration_tests/start.sh"]