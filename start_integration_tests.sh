docker build -f ./integration_tests/Dockerfile -t 'cudos-ondemand-minting-service' .

docker run --name cudos-ondemand-minting-service cudos-ondemand-minting-service
