# CUDOS Ondemand Minting Service

## Config
`wallet_mnemonic:` - Mnemonic that will be managed by the service and used to mint the NFTs.  
`chain:` - GRPC, RPC and chain id of the network.  
`tokenised_infra_url:` - Url to API that provides the NFT data.  
`state_file: ` - Filename where state of service will be stored, currently this is only the last process height.   
`max_retries: ` - If service fails during processing of some requests, this is the maximum number of retries before the service exits.  
`retry_interval: ` - Delay between retries.   
`relay_interval: ` - Interval at which the service will check for requests to process.  
`payment_denom: ` - Payment denom used by the network and requests.

## Starting the service:

Build the docker image:\
```docker build -t 'cudos-ondemand-minting-service' .```

Run the docker image:\
```docker run -d --name cudos-ondemand-minting-service cudos-ondemand-minting-service```
