# Gateway

## Instructions

1. Make sure golang is installed
2. You know the path to your `docker.sock` (defaults to /var/run/docker)
3. `docker-compose up` to bring up pre-labeled nginx containers
4. `make build && make run (served on 8080)`
5. `GET - /stats` for telemetry

## Design

![](https://github.com/donnemartin/system-design-primer/raw/master/images/h81n9iK.png)

- Selection Algorithm: uses minHeap to keep track of weights and priority of the upstream connections.
- Inmem implementation, although the `repo` could be replaced with a `redisCli` that matches the repo interface.

## Future Considerations

- I would like to have added more testing, but did not have the time.
- I would use prometheus or other proven telemetry clients, instead of a quick version of my own.
- I would like to have detected if the host network is bound to the docker-network or eth0
- I would like to have added HEAD health checks.
