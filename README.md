# gowxapi

# Local Env
This API uses ArcGIS service to get lat/long from a given city or state name. Create a free API key on ArcGIS developer website use that key in this API.

How to get the environment variables, either one of these is acceptable,

### Create a local `.env` file and add this content with proper API KEY.
```
API_KEY=ARCGIS_API_KEY
GIN_MODE=release
```

### Create environment variables in your IDE.
```
API_KEY=ARCGIS_API_KEY
GIN_MODE=release
```
NOTE: These environment variables are required in your cloud/remote environment as well. For example, in Azure, these variables should be added as secrets in your resource running this API.