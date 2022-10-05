WORKDIR=$PWD
BRANCH='cudos-dev'

git clone -b $BRANCH https://github.com/CudoVentures/cudos-node.git $WORKDIR/cudos-node
git clone -b $BRANCH https://github.com/CudoVentures/cudos-builders.git $WORKDIR/cudos-builders

cd $WORKDIR/cudos-node
make install

cd $WORKDIR
cp $WORKDIR'/root-node.local.env' $WORKDIR'/cudos-builders/docker/root-node/root-node.local.env'

sed -i 's/keyring-backend os/keyring-backend test/g' $WORKDIR'/cudos-builders/docker/root-node/scripts/init-root.sh'

cd $WORKDIR'/cudos-builders/tools-nodejs/init-local-node-without-docker'

chmod +x init.sh
source ./init.sh 2>node.output &

cd $WORKDIR