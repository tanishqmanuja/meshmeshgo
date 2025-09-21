# Running with Docker

This guide explains how to run the meshmeshgo program in Docker. The app requires access to:
1. Serial device (e.g., /dev/ttyUSB0)
2. User-provided files:
    - meshmeshgo.json
    - meshmesh.graphml


## 1. Build the Docker Image

From the project root (where the Dockerfile is located):

```bash
docker build -t meshmeshgo .
```


## 2.a. Run with `docker run`

```bash
docker run --rm -it \
  --network host \
  --privileged \
  -v $(pwd)/meshmeshgo.json:/root/meshmeshgo.json:ro \
  -v $(pwd)/meshmesh.graphml:/root/meshmesh.graphml:ro \
  meshmeshgo
```


### 2.b. Run using `Docker Compose`:

```bash
docker compose up --build
```

* `--build` ensures the image rebuilds if source changes.
* you can run without --build after first time

