# THIS IS CURRENTLY EXPERIMENTAL!

# About 

This is a *simple* HTTP => statsd for quickly tracking metrics. 

## Sending Metrics Data

There are four endpoints for tracking metrics: gauge, count, history and set. These map directly to 
[Datadog's metrics](http://docs.datadoghq.com/guides/dogstatsd/#metrics).


````
  curl -v -d '123.45' https://<endpoint>/gauge/<metric name>
  curl -v -d '123' https://<endpoint>/count/<metric name>
  curl -v -d '123.45' https://<endpoint>/histogram/<metric name>
  curl -v -d 'some string' https://<endpoint>/set/<metric name>
````

## Running it

* Currently expects the datadog agent to exist on localhost:8125
* Configuration is through environment variables.
* `LISTEN` - the address to listen on, defaults to ":8080".
* `NAMESPACE` - what to automatically prefix all metrics with. Defaults to `experimental.`. Name must end with a period.
* `TAGS` - a comma separated list of tags to send along with *all* metrics
* `WHITELIST_FILE` - path a list of metrics to allow. Not specifying this will accept any metric name. An empty whitelist file accepts nothing.

## License

[MPL v 2.0](https://www.mozilla.org/MPL/2.0/index.txt)
