## TinyRP - Simple lightweight HTTP proxy

TinyRP is a simple lightweight HTTP reverse proxy made in golang


### Current available feature

[X] reverse proxy based on endpoint 


### Usage

1. Create a config file in this format

```
resources:
  - name: Server1
    endpoint: /server1
    destination_url: "http://localhost:9001"
  - name: Server2
    endpoint: /server2
    destination_url: "http://localhost:9002"
  - name: Server3
    endpoint: /server3
    destination_url: "http://localhost:9003"

```

2. Run reverse proxy and pass config file as an argument as shown below

```
tinyrp --path="PATH_TO_RESOURCE_FILE"
```

##### Run demo services

The below command will run three demo services server1, server2 and server3

```
make run
```


### Requirement

1. Docker