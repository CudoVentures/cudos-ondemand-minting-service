CHAIN_ID='cudos-local-network'
FEE_FLAGS='--gas auto --gas-adjustment 1.3 --gas-prices 5000000000000acudos'
WALLET_ADDRESS='cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv'
WORKDIR=$PWD
CUDOS_NODED_RUNNING_INSTANCE_PATH=$WORKDIR'/integration_tests/cudos-builders/tools-nodejs/init-local-node-without-docker'
EXPECTED_COVERAGE='100.0'

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

# ADD MARKETPLACE ADMIN
echo "ADD MARKETPLACE ADMIN"
cudos-noded tx marketplace add-admin $marketplaceAdminAddress --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS --from=marketplace-admin-account -y

# VERIFY COLLECTION
echo "VERIFY COLLECTION"
cudos-noded tx marketplace verify-collection 0 --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS --from=marketplace-admin-account -y

# BANK SEND WITH VALID MEMO
echo "BANK SEND WITH VALID MEMO"
cudos-noded tx bank send minting-tester "$WALLET_ADDRESS" 9000000000000000000acudos --note="{\"uuid\":\"nftuid1\"}" --keyring-backend test --chain-id="$CHAIN_ID" $FEE_FLAGS -y

cd $WORKDIR

echo "WAIT FOR INTEGRATION TESTS TO COMPLETE"
# TODO: Have some smarter way to wait for tests to complete
sleep 20

echo "STOP INTEGRATION TESTS"
curl http://127.0.0.1:19999

echo "START UNIT TESTS"
go test -timeout 30s -v -cover -covermode=count -coverprofile unittests1.out -run ^TestShouldExitIfInvalidConfigFilename$ ./cmd/cudos-ondemand-minting-service
go test -timeout 30s -v -cover -covermode=count -coverprofile unittests2.out -run ^TestShouldExitIfConfigContainsInvalidMnemonic$ ./cmd/cudos-ondemand-minting-service
go test -timeout 30s -v -cover -covermode=count -coverprofile unittests3.out ./internal/...

GO111MODULE=off go get github.com/wadey/gocovmerge
gocovmerge *.out > merged.cov
go tool cover -func=merged.cov | grep -E '^total\:' | sed -E 's/\s+/ /g'

COVERAGE=$(go tool cover -func merged.cov | grep total | awk '{print substr($3, 1, length($3)-1)}')

echo "Coverage is $COVERAGE"
COVERAGE_INT=$(printf "%.0f" "$COVERAGE")
if [ "$COVERAGE_INT" -lt "90" ];then
    echo "Expected coverage is $EXPECTED_COVERAGE but actual is $COVERAGE"
    exit 1
fi

echo "Successfully passed tests!"
