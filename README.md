# Weather API
#### written in Go ([golang](https://go.dev))
[![Publish](https://github.com/hnabbasi/gowxapi/actions/workflows/publish.yml/badge.svg?branch=main)](https://github.com/hnabbasi/gowxapi/actions/workflows/publish.yml)

This API aggregates over 10 National Weather Service APIs to provide concise information in a usable manner.
# Local Env
This API uses ArcGIS service to get lat/long from a given city or state name. Create a free API key on [ArcGIS developer portal](https://developers.arcgis.com) to use with this API.
### Environment variables needed
```
API_KEY=ARCGIS_API_KEY
GIN_MODE=release
```
These environment variables are required in your cloud/remote environment as well. For example, in Azure, these variables should be added as secrets in your resource running this API.