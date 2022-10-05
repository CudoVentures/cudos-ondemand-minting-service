WORKDIR=$PWD
BRANCH='cudos-dev'

git clone -b $BRANCH https://github.com/CudoVentures/cudos-node.git $WORKDIR/cudos-node
git clone -b $BRANCH https://github.com/CudoVentures/cudos-builders.git $WORKDIR/cudos-builders

cd $WORKDIR/cudos-node
make install

cd $WORKDIR
cp $WORKDIR'/root-node.local.env' $WORKDIR'/cudos-builders/docker/root-node/root-node.local.env'

cd $WORKDIR'/cudos-builders/tools-nodejs/init-local-node-without-docker'

chmod +x init.sh
source ./init.sh 2>node.output &

echo "WILL GET SOME SLEEP"
sleep 30
echo "WILL CHECK THE OUTPUT"

echo "CUDOS HOME OUTPUT"
echo $CUDOS_HOME

echo "WILL OUTPUT APP.TOML CONFIG"
cat ./cudos-data/config/app.toml

echo "WILL OUTPUT APP.TOML CONFIG VIA CUDOS_HOME"
cat $CUDOS_HOME'/cudos-data/config.toml'

cd $WORKDIR