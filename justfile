export IMAGE_PATH := "headgardener/docker-reload"

check:
	docker compose --file example/compose.yml down
	docker compose --file example/compose.yml up --no-start --build
	touch example/watchdir/file
	docker compose --file example/compose.yml up -d
	docker compose --file example/compose.yml logs -f &
	sleep 30
	docker compose --file example/compose.yml down

build:
  docker build --build-arg DOCKER_VERSION=v25.0.8+incompatible -t "$IMAGE_PATH:v25" .
  docker build --build-arg DOCKER_VERSION=v26.1.5+incompatible -t "$IMAGE_PATH:v26" .
  docker build --build-arg DOCKER_VERSION=v27.5.1+incompatible -t "$IMAGE_PATH:v27" .
  docker build --build-arg DOCKER_VERSION=v28.0.1+incompatible -t "$IMAGE_PATH:v28" .

push:
  docker tag "$IMAGE_PATH:v27" "$IMAGE_PATH:latest"
  echo -n "latest v25 v26 v27 v28" | xargs -d' ' -tI {} docker push "$IMAGE_PATH:{}"

push-release:
  #!/usr/bin/env bash
  set -ex
  tag="$(git describe --tags --exact-match)"
  echo -n "v25 v26 v27 v28" | xargs -d' ' -tI {} docker tag "$IMAGE_PATH:{}" "$IMAGE_PATH:$tag-{}"
  docker tag "$IMAGE_PATH:v27" "$IMAGE_PATH:latest"
  docker tag "$IMAGE_PATH:v27" "$IMAGE_PATH:$tag"
  echo -n "latest $tag v25 v26 v27 v28 $tag-v25 $tag-v26 $tag-v27 $tag-v28" \
    | xargs -d' ' -tI {} docker push "$IMAGE_PATH:{}"
