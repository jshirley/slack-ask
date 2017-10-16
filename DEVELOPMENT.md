# Use a docker based Mongo

Run Mongo, this just pulls [the Mongo image](https://hub.docker.com/_/mongo/):

`docker run --name slack-ask-mongo -p 27017:27017 -d mongo`

If you want to connect to the mongo to inspect:

`docker run -it --link slack-ask-mongo:mongo --rm mongo sh -c exec mongo "$MONGO_PORT_27017_TCP_ADDR:$MONGO_PORT_27017_TCP_PORT/test"`

# Use Docker for running slack-ask

```
docker build ./
docker run
```

Disclaimer, I don't really know how to use Docker correctly. Plz help.

