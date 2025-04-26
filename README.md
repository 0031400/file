### summary
This is a simple golang filebed.It has two function.Upload and get the file.It has a simple basic auth.
### config
the config file is `config.yaml`  
See the config.yaml to see how to config.  
You need to write down all config items to make sure it work properly.
### api
- request: `/upload` post  
body: form-data `file` field  
response: url like `i/2025/04/26/81917c11-18fa-4aaf-9111-f4ddcafdef8a.png`
- request `/{path}` get  
path like `i/2025/04/26/81917c11-18fa-4aaf-9111-f4ddcafdef8a.png`  
body: the file
### auth
the `/upload` need basic auth
### log
no
### cache
no but it has the cache header 100y