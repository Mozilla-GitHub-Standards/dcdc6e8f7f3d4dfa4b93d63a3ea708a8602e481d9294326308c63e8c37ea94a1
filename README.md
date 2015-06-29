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

## License

[MPL v 2.0](https://www.mozilla.org/MPL/2.0/index.txt)
