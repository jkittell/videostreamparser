docker login

docker buildx create

docker buildx build --push \
--provenance false \
--platform linux/arm64/v8,linux/amd64 \
--tag jpkitt/videostreamsegments:latest .
