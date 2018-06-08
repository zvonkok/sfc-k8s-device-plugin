cd container

docker build  -t sfc-dev-plugin  .
docker tag sfc-dev-plugin quay.io/zvonkok/sfc-dev-plugin
docker push quay.io/zvonkok/sfc-dev-plugin


cd ..
