# Build and Publish the Collector Image

Log in to Docker Hub:

```sh
docker login
```

Create and use a multi-platform builder once:

```sh
docker buildx create --name multiarch --driver docker-container --use
docker run --privileged --rm tonistiigi/binfmt --install all
```

Build and push with the current commit SHA as the tag:

```sh
TAG="<your-dockerhub-account>/grafanacloud-collector:$(git rev-parse --short HEAD)"
docker buildx build --platform linux/amd64,linux/arm64 -t "$TAG" --push .
```

If the multi-platform build is too resource intensive, publish amd64 only:

```sh
TAG="<your-dockerhub-account>/grafanacloud-collector:$(git rev-parse --short HEAD)"
docker buildx build --platform linux/amd64 -t "$TAG" --push .
```
