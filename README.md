# 🌻 Achoo – Pollen Forecast API for Germany

This is the source code for the API server running over at [https://api.achoo.dev](https://api.achoo.dev). Check out the documentation at [https://achoo.dev](https://achoo.dev).

## Where is the data coming from?

The API is using data provided by the Deutscher Wetterdienst (DWD). The raw data can be found [here](https://opendata.dwd.de/climate_environment/health/alerts/s31fg.json). Basically all I'm doing is slightly changing the format and providing endpoints to query only parts of the data.

## Running the server

Build the project by either running `make dist` (builds for linux/arm by default) or by running `go build -o dist/pollen-api`.

Start the server by executing the binary

```
./pollen-api
```

This will start an HTTP server listening on port 8000. The server itself does not support HTTPS, so you should use a reverse proxy for that.

### Redis

The server stores all its data in redis. You can configure the connection parameters through environment variables.

| Variable           | Description                                         | Default          |
| :----------------- | :-------------------------------------------------- | :--------------- |
| `REDIS_HOST`       | The address including the port of the redis server. | `localhost:6379` |
| `REDIS_KEY_PREFIX` | If set, all redis keys will be prefixed with this.  | `""`             |
| `REDIS_PASSWORD`   | Password to use when connecting to the redis erver. | `""`             |

## Running the tests

`make test`

## License

MIT
