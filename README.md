# Bot Wrangler Traefik Plugin

> [!WARNING]  
> This project is still in active development and is not ready to be included through the official plugin catalog.

- [Bot Wrangler Traefik Plugin](#bot-wrangler-traefik-plugin)
- [Features](#features)
- [Configuration](#configuration)
  - [Local Mode](#local-mode)
- [Credits](#credits)

Bot Wrangler is a Traefik plugin designed to improve your web application's security and performance by managing bot traffic effectively. With the rise of large language model (LLM) data scrapers, it has become crucial to control automated traffic from bots. Bot Wrangler provides a solution to log and/or block traffic from unwanted LLM bots, ensuring that your resources are protected and your data remains secure.

LLM Bot user agents are retrieved from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt). Any queries to a service where this middleware is implemented will provide this list when `/robots.txt` is queried. If a request is **not** to `/robots.txt`, but is from a bot in the LLM bot list, the user agent may be logged (and blocked if desired) with a 403 response.

# Features

- Dynamic Updates: Automatically fetches and applies the latest LLM Bot user-agents rules from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt)
- Traffic Logging: Keep track of bot traffic for analysis and reporting
- Unified Bot Management: No longer maintain a robots.txt file for each of your applications
- Customizable Settings: Tailor the plugin's behavior to suit your specific needs and preferences, such as a custom bots list.

# Configuration

For each plugin, the Traefik static configuration must define the module name (as is usual for Go packages).

For this plugin, it will look like the following:

```yaml
# Static configuration
experimental:
  plugins:
    bouncer:
      moduleName: github.com/holysoles/bot-wrangler-traefik-plugin
      version: vX.Y.Z # find latest release
```

Here is an example of a file provider dynamic configuration (given here in YAML), where the interesting part is the `http.middlewares` section:

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

Traefik also offers a developer mode that can be used for temporary testing of plugins not hosted on GitHub.
To use a plugin in local mode, the Traefik static configuration must define the module name (as is usual for Go packages) and a path to a [Go workspace](https://golang.org/doc/gopath_code.html#Workspaces), which can be the local GOPATH or any directory.

A test dev instance can be easily setup by using commands in the provided makefile (e.g. `make run_local`, `make restart_local`) and modifying the `docker-compose.local.yml` file.

# Credits

Special thanks to the following projects:
- [ai.robots.txt](https://github.com/ai-robots-txt/ai.robots.txt)
- [Ashley McNamara](https://github.com/ashleymcnamara/gophers) for the icon