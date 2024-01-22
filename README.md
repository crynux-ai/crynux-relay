# h-relay
The layer 2 relay server for the Crynux node


### Running the tests
The tests could be executed using a docker image.

1. Build the docker image using the Dockerfile given under the ```tests``` package:

```shell
# docker build -t crynux_relay:test -f .\tests\test.Dockerfile .
```

2. Run the tests:

```shell
# docker run -it --rm crynux_relay:test 
```
