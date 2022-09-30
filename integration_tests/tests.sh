CHAIN_ID='cudos-local-network'
FEE_FLAGS='--gas auto --gas-adjustment 1.3 --gas-prices 5000000000000acudos'
WALLET_ADDRESS='cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv'
CUDOS_NODED_RUNNING_INSTANCE_PATH='/Users/angelvalkov/git/cudos-node'
CURRENT_DIR=$PWD

# COMPILE MOCK SERVICE
go build ./cmd/mock-service

# EXECUTE MOCK SERVICE
./mock-service &

# COMPILE MINTING SERVICE
go test -c ./cmd/cudos-ondemand-minting-service -cover -covermode=count -coverpkg=./...

# EXECUTE MINTING SERVICE
./cudos-ondemand-minting-service.test -test.coverprofile cudos-ondemand-minting-service.out &

cd $CUDOS_NODED_RUNNING_INSTANCE_PATH

# CREATE USERS
echo "CREATE USERS"
echo yes | cudos-noded keys add minting-tester --keyring-backend test

mintingTesterAddress=$(cudos-noded keys show -a minting-tester --keyring-backend test)
marketplaceAdminAddress=$(cudos-noded keys show -a marketplace-admin-account --keyring-backend test)

# ADD FUNDS
echo "ADD FUNDS"
cudos-noded tx bank send faucet "$mintingTesterAddress" 100000000000000000000acudos --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS -y
cudos-noded tx bank send faucet "$marketplaceAdminAddress" 100000000000000000000acudos --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS -y

# ISSUE DENOM
echo "ISSUE DENOM"
cudos-noded tx nft issue testdenom --name=testdenom --symbol=testdenom --minter="$WALLET_ADDRESS" --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS --from=minting-tester -y

# PUBLISH COLLECTION FOR SALE
echo "PUBLISH COLLECTION FOR SALE"
cudos-noded tx marketplace publish-collection testdenom --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS --from=minting-tester -y

# VERIFY COLLECTION
echo "VERIFY COLLECTION"
cudos-noded tx marketplace verify-collection 0 --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS --from=marketplace-admin-account -y

# BANK SEND WITH VALID MEMO
echo "BANK SEND WITH VALID MEMO"
cudos-noded tx bank send minting-tester "$WALLET_ADDRESS" 9000000000000000000acudos --note="{\"uid\":\"nftuid1\"}" --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS -y

cd $CURRENT_DIR

echo "WAIT FOR INTEGRATION TESTS TO COMPLETE"
sleep 20

echo "STOP INTEGRATION TESTS"
curl http://127.0.0.1:19999

echo "START UNIT TESTS"
#TODO: Move unit tests on top and have here some realiable way to know that the minting service exited
go test -timeout 30s -v -cover -covermode=count -coverprofile unittests.out -run ^TestRelayMinter$ ./internal/relay_minter
#GO111MODULE=off go get github.com/wadey/gocovmerge
gocovmerge *.out > merged.cov
go tool cover -func=merged.cov | grep -E '^total\:' | sed -E 's/\s+/ /g'
