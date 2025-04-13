# Bot Wrangler Traefik Plugin

![GitHub License](https://img.shields.io/github/license/holysoles/bot-wrangler-traefik-plugin)
[![codecov](https://codecov.io/gh/holysoles/bot-wrangler-traefik-plugin/graph/badge.svg?token=1GCKDQSR7R)](https://codecov.io/gh/holysoles/bot-wrangler-traefik-plugin)
![Issues](https://img.shields.io/github/issues/holysoles/bot-wrangler-traefik-plugin)

Bot Wrangler is a Traefik plugin designed to improve your web application's security and performance by managing bot traffic effectively. With the rise of large language model (LLM) data scrapers, it has become crucial to control automated traffic from bots. Bot Wrangler provides a solution to log, block, or otherwise handle traffic from unwanted LLM bots, ensuring that your resources are protected and your content remains accessibility only to those desired.

LLM Bot user agents are retrieved from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt). Any queries to a service where this middleware is implemented will provide this list when `/robots.txt` is queried. If a request is from a bot in the LLM bot list, meaning its ignoring `robots.txt`, a configurable remediation action is taken:

- `PASS`: Do nothing (no-op)
- `LOG`: write a log message about the visitor, the default behavior
- `BLOCK`: reject the request with a 403 error
- `PROXY`: proxy the request to a "tarpit" or other service to handle bot traffic, such as [Nepenthes](https://zadzmo.org/code/nepenthes/), [iocaine](https://iocaine.madhouse-project.org), etc

## Table Of Contents

- [Bot Wrangler Traefik Plugin](#bot-wrangler-traefik-plugin)
  - [Table Of Contents](#table-of-contents)
- [Features](#features)
- [Usage](#usage)
  - [Considerations](#considerations)
  - [Configuration](#configuration)
    - ["Tarpits" to Send Bots to](#tarpits-to-send-bots-to)
  - [Deployment](#deployment)
    - [Generic](#generic)
    - [Kubernetes](#kubernetes)
    - [Local/Dev Mode](#localdev-mode)
- [Contributions](#contributions)
- [Credits](#credits)

# Features

- Dynamic Updates: Automatically fetches and applies the latest LLM Bot user-agents rules from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt)
- Traffic Logging: Keep track of bot traffic for analysis and reporting
- Unified Bot Management: No longer maintain a robots.txt file for each of your applications
- Customizable Settings: Tailor the plugin's behavior to suit your specific needs and preferences, such as a custom bots list.

# Usage

## Considerations

Please read if you plan to deploy this plugin!

- Ensure that the `robots.txt` template is available to Traefik at startup. For Docker, this means passing the file in as a mount. For Kubernetes, mounting a the template in a ConfigMap is easiest.
- The reverse proxy to the `botProxyUrl` is unbuffered. If you are passing this request to _another_ reverse proxy in front of a tarpit-style application, ensure proxy buffering is disabled.
- The cache from ai-robots-txt is refreshed if expired during a request. While this is set (by default) to update every 24 hours, there will be a small response speed impact (<0.06s) on the request that causes the cache refresh.

## Configuration

The follow parameters are exposed to configure this plugin

| Name | Default Value | Description |
|------|---------------|-------------|
|enabled|`true`|Whether or not the plugin should be enabled|
|cacheUpdateInterval|`24h`|How frequently the robots list should be updated|
|botAction|`LOG`|How the bot should be wrangled. Available: `PASS` (do nothing), `LOG` (log bot info), `BLOCK` (log and return 403), `PROXY` (log and proxy to `botProxyUrl`)|
|botProxyUrl|``|The URL to pass a bot's request to, if `PROXY` is the set `botAction`|
|logLevel|`INFO`|THe log level for the plugin|
|robotsTxtFilePath|`robots.txt`| The file path to the robots.txt template file. You can customize the provided file as desired|
|robotsSourceUrl|`https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.json`|A URL to a JSON formatted robot user agent index. You can provide your own, but ensure it has the same JSON keys!|

### "Tarpits" to Send Bots to

There are many applications that folks have wrote that are meant to handle LLM in traffic in some way to waste their time, usually based off Markov Chains, or even a local LLM instance to generate some random text. Some you need to provide training data to, some are already trained. Some are more malicious in nature than others, so deploy at your own risk!

I have not tested this plugin with this list, nor is it an exhaustive list of all projects in this space. If you find this plugin has issues with one, please open an issue. Thanks to iocaine for providing this initial list!

  - [iocaine](https://iocaine.madhouse-project.org/)
  - [Nepenthes](https://zadzmo.org/code/nepenthes/)
  - [Quixotic](https://marcusb.org/hacks/quixotic.html)
  - [marko](https://codeberg.org/timmc/marko/)
  - [Poison the WeLLMs](https://codeberg.org/MikeCoats/poison-the-wellms)
  - [django-llm-poison](https://github.com/Fingel/django-llm-poison)
  - [konterfai](https://codeberg.org/konterfai/konterfai)
  - [caddy-defender](https://github.com/JasonLovesDoggo/caddy-defender)
  - [markov-tarpit](https://git.rys.io/libre/markov-tarpit)
  - [spigot](https://github.com/gw1urf/spigot)

## Deployment

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

### Generic

After including the plugin module in Traefik's static configuration, you'll need to setup the dynamic configuration to actually use it.

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

### Kubernetes

1. Include Plugin in Traefik configuration

If Traefik is deployed with the official helm chart, you'll need to include these values in your Values.yaml for the release:

```yaml
experimental:
  plugins:
    wrangler:
      moduleName: "github.com/holysoles/bot-wrangler-traefik-plugin"
      version: vX.Y.Z # find latest release here: https://github.com/holysoles/bot-wrangler-traefik-plugin/releases
volumes:
  - name: botwrangler-robots-template
    mountPath: /etc/traefik/bot-wrangler/
    type: configMap
```

2. Create configMap for robots.txt template

We'll need to create the configMap being referenced (ensure its in the same namespace!):

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: botwrangler-robots-template
data:
  robots.txt: |
    {{ range $agent := .UserAgentList }}
    User-agent: {{ $agent }}
    {{- end }}
    Disallow: /
```

3. Define Middleware

Then we'll need to create the middleware object:

```yaml
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: botwrangler
spec:
  plugin:
    wrangler:
      robotsTxtFilePath: /etc/traefik/bot-wrangler/robots.txt
      # Any other config options go here
```

4. Apply the plugin

As for actually **including** the middleware, you can either include the middleware per-IngressRoute:

```yaml
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: whoami-ingress
spec:
  entryPoints:
  - web
  routes:
  - kind: Rule
    match: Host(`example.com`)
    middlewares:
    - name: botwrangler
    services:
    - name: whoami
      port: 8080
```

Or per-entrypoint (in your Values.yaml) if you want to broadly apply the plugin:

```yaml
    additionalArguments:
      - "--entrypoints.web.http.middlewares=traefik-botwrangler@kubernetescrd"
      - "--entrypoints.websecure.http.middlewares=traefik-botwrangler@kubernetescrd"
      - "--providers.kubernetescrd"
```

### Local/Dev Mode

Traefik offers a developer mode that can be used for temporary testing of unpublished plugins.

To use a plugin in local mode, the Traefik static configuration must define the module name (as is usual for Go packages) and a path to a [Go workspace](https://golang.org/doc/gopath_code.html#Workspaces), which can be the local GOPATH or any directory.

A test dev instance can be easily setup by using commands in the provided makefile (e.g. `make run_local`, `make restart_local`) and modifying the `docker-compose.local.yml` file.

# Contributions

Contributions to this project are welcome! Please use conventional commits, and retain a linear git history.

# Credits

Special thanks to the following projects:
- [ai.robots.txt](https://github.com/ai-robots-txt/ai.robots.txt) as an essential information source for this tool
- [Ashley McNamara](https://github.com/ashleymcnamara/gophers) for the icon!
- [crowdsec-bouncer-traefik-plugin](https://github.com/maxlerebourg/crowdsec-bouncer-traefik-plugin) for inspiration as a well-written Traefik plugin