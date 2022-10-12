# CUDOS Ondemand Minting Service

Purpose of this service is to manage wallet that is whitelisted for minting given denom. Non-whitelisted users will send bank send message to this wallet amounting to the price of NFT plus fees to mint it and UUID of the NFT in the database as the transaction memo.

The way to whitelist wallet is to use the ```--minter``` flag during issuing of the denom.

```cudos-noded tx nft issue testdenom1 --name=testdenom1 --symbol=testdenom1 --minter="cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv" --keyring-backend os --chain-id="cudos-dev-test-network" --gas auto --gas-adjustment 1.3 --gas-prices 5000000000000acudos --from=minting-tester```

The service will be constantly checking for transactions to the managed wallet. Once it encounters such it will check if it has single bank send message inside with memo containing UUID. Incase someone like malicious actor tries to play with the service by sending some other transactions, we will skip them and will not refund him.

Command to send funds in tx with UUID in the memo:
```cudos-noded tx bank send minting-tester cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv 9000000000000000000acudos --note="{\"uuid\":\"nftuid1\"}" --keyring-backend os --chain-id="cudos-dev-test-network" --gas auto --gas-adjustment 1.3 --gas-prices 5000000000000acudos```

If we have valid transaction, we will check if the NFT with this UUID is not minted already by checking the events onchain, if its minted, then we will refund the user by subtracting the refund tx fee from the funds that he sent to us. If its not minted we will fetch the full NFT data via the aura pay backend and mint it via the marketplace.

After processing transaction successfully (either skip/refund/mint) it will increase the last process block height which is stored in ```state.json```, which is just optimization to scan only from this high above.