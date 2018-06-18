# Json Exporter
![TravisCI build status](https://travis-ci.org/tolleiv/json-exporter.svg?branch=master)
[![Docker Build Statu](https://img.shields.io/docker/build/tolleiv/json-exporter.svg)](https://hub.docker.com/r/tolleiv/json-exporter/)

This Prometheus exporter operates similar to the Blackbox exporters. It downloads a JSON file and provides a numerical gauge value from within that file.
Which values to pick is defined through jsonpath.[METRIC_NAME_M]=[JSON PATH_N].

## Parameters

 - `target`: URL / Json-file to download
 - `jsonpath.[METRIC_NAME_1]=[JSON PATH_1]`: the field name to read the value from, this follows the syntax provided by [oliveagle/jsonpath](https://github.com/oliveagle/jsonpath)
 - `jsonpath.[METRIC_NAME_N]=[JSON PATH_N]`: the field name to read the value from, this follows the syntax provided by [oliveagle/jsonpath](https://github.com/oliveagle/jsonpath)

## Docker usage

    docker build -t json_exporter .
    docker -d -p 9116:9116 --name json_exporter json_exporter
   
The related metrics can then be found under:
   
    http://localhost:9116/probe?target=http://validate.jsontest.com/?json=%7B%22key%22:%22value%22%7D&jsonpath.nanoseconds=$.parse_time_nanoseconds&jsonpath.size=$.size

## Prometheus Configuration

The json exporter needs to be passed the target and the json as a parameter, this can be
done with relabelling.

Example config:
```yml
scrape_configs:
  - job_name: 'json'
    metrics_path: /probe
    params:
      - jsonpath.nanoseconds: [$.parse_time_nanoseconds] # Look for the nanoseconds field
      - jsonpath.size: [$.size]
    static_configs:
      - targets:
        - http://validate.jsontest.com/?json=%7B%22key%22:%22value%22%7D
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: 127.0.0.1:9116  # Json exporter.

```

## License

MIT License
