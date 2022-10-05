exit 1
cd ./integration_tests

source ./setup_node.sh

# TODO: Have smarter way to wait make sure that node is running
sleep 30

cd ..
source ./integration_tests/tests.sh
source ./integration_tests/cleanup.sh