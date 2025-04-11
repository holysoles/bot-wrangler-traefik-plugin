# Bot Wrangler Traefik Plugin

- [Bot Wrangler Traefik Plugin](#bot-wrangler-traefik-plugin)
- [Features](#features)
- [Usage](#usage)
  - [Considerations](#considerations)
  - [Configuration](#configuration)
  - [Configuration Example](#configuration-example)
  - [Local Mode](#local-mode)
- [Credits](#credits)

Bot Wrangler is a Traefik plugin designed to improve your web application's security and performance by managing bot traffic effectively. With the rise of large language model (LLM) data scrapers, it has become crucial to control automated traffic from bots. Bot Wrangler provides a solution to log and/or block traffic from unwanted LLM bots, ensuring that your resources are protected and your content remains accessibility only to those desired.

LLM Bot user agents are retrieved from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt). Any queries to a service where this middleware is implemented will provide this list when `/robots.txt` is queried. If a request is from a bot in the LLM bot list, meaning its ignoring `robots.txt`, the user agent may be logged (and blocked if desired) with a 403 response.

# Features

- Dynamic Updates: Automatically fetches and applies the latest LLM Bot user-agents rules from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt)
- Traffic Logging: Keep track of bot traffic for analysis and reporting
- Unified Bot Management: No longer maintain a robots.txt file for each of your applications
- Customizable Settings: Tailor the plugin's behavior to suit your specific needs and preferences, such as a custom bots list.

# Usage

## Considerations

Please read if you plan to deploy this plugin!

- The cache from ai-robots-txt is refreshed if expired during a request. While this is set (by default) to update every 24 hours, there will be a small response speed impact (<0.06s) on the request that causes the cache refresh.

## Configuration

The follow parameters are exposed to configure this plugin

| Name | Default Value | Description |
|------|---------------|-------------|
|cacheUpdateInterval|`24h`|How frequently the robots list should be updated|
|botAction|`LOG`|How the bot should be wrangled. Available: `PASS` (do nothing), `LOG` (log bot info), `BLOCK` (log and return 403)|
|logLevel|`INFO`|THe log level for the plugin|
|robotsTxtFilePath|`robots.txt`| The file path to the robots.txt template file. You can customize the provided file as desired|
|robotsSourceUrl|`https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.json`|A URL to a JSON formatted robot user agent index. You can provide your own, but ensure it has the same JSON keys!|

## Configuration Example

The Traefik static configuration must define the module name:

```yaml
# Static configuration
experimental:
  plugins:
    wrangler:
      moduleName: github.com/holysoles/bot-wrangler-traefik-plugin
      version: vX.Y.Z # find latest release here: https://github.com/holysoles/bot-wrangler-traefik-plugin/releases
```

For actually including the plugin as middleware, you'll need to include it in Traefik's dynamic configuration. 

Here is an example of a file provider dynamic configuration (given here in YAML). note the `http.routers.my-router.middlewares` and `http.middlewares` sections:

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - bot-wrangler

  services:
   service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000
  
  middlewares:
    bot-wrangler:
      plugin:
        wrangler:
          logLevel: INFO
          botAction: BLOCK
```

## Local Mode

Traefik offers a developer mode that can be used for temporary testing of unpublished plugins.

To use a plugin in local mode, the Traefik static configuration must define the module name (as is usual for Go packages) and a path to a [Go workspace](https://golang.org/doc/gopath_code.html#Workspaces), which can be the local GOPATH or any directory.

A test dev instance can be easily setup by using commands in the provided makefile (e.g. `make run_local`, `make restart_local`) and modifying the `docker-compose.local.yml` file.

# Credits

Special thanks to the following projects:
- [ai.robots.txt](https://github.com/ai-robots-txt/ai.robots.txt) as an essential information source for this tool
- [Ashley McNamara](https://github.com/ashleymcnamara/gophers) for the icon