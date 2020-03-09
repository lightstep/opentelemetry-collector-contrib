# LightStep Exporter

This exporter sends OpenTelemetry traces to LightStep.

The following configuration options are supported:

- `access_token` (Required): Your LightStep access token used to identify the project
associated with the configured collector.
- `satellite_url` (Optional): The URL of your LightStep satellites. If not set, defaults
to https://ingest.lightstep.com:443.

Example:

```yaml
exporters:
  lightstep:
    access_token: "access-token"
    satellite_url: "https://ingest.lightstep.com:443"
```