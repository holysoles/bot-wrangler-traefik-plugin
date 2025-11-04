# Bot Wrangler Traefik Plugin

![GitHub License](https://img.shields.io/github/license/holysoles/bot-wrangler-traefik-plugin)
[![codecov](https://codecov.io/gh/holysoles/bot-wrangler-traefik-plugin/graph/badge.svg?token=1GCKDQSR7R)](https://codecov.io/gh/holysoles/bot-wrangler-traefik-plugin)
[![Go Report Card](https://goreportcard.com/badge/github.com/holysoles/bot-wrangler-traefik-plugin)](https://goreportcard.com/report/github.com/holysoles/bot-wrangler-traefik-plugin)
![Issues](https://img.shields.io/github/issues/holysoles/bot-wrangler-traefik-plugin)

Bot Wrangler is a Traefik plugin designed to improve your web application's security and performance by managing bot traffic effectively. With the rise of large language model (LLM) data scrapers, it has become crucial to control automated traffic from bots. Bot Wrangler provides a solution to log, block, or otherwise handle traffic from unwanted LLM bots, ensuring that your resources are protected and your content remains accessible only to those desired.

By default, Bot user agents are retrieved from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt). Any queries to a service where this middleware is implemented will provide this list when `/robots.txt` is queried. If an incoming request matches the bot list, meaning it is ignoring your `robots.txt`, a configurable remediation action is taken:

- `PASS`: Do nothing (no-op)
- `LOG`: write a log message about the visitor, the default behavior
- `BLOCK`: reject the request with a static response (a 403 error by default)
- `PROXY`: proxy the request to a "tarpit" or other service to handle bot traffic, such as [Nepenthes](https://zadzmo.org/code/nepenthes/), [iocaine](https://iocaine.madhouse-project.org), etc

## Table Of Contents

<!-- toc -->

- [Features](#features)
- [Usage](#usage)
  * [Considerations](#considerations)
  * [Configuration](#configuration)
    + [Providing Custom Robots Sources](#providing-custom-robots-sources)
    + ["Tarpits" to Send Bots to](#tarpits-to-send-bots-to)
  * [Deployment](#deployment)
    + [Generic](#generic)
    + [Kubernetes](#kubernetes)
    + [Local/Dev Mode](#localdev-mode)
- [Contributions](#contributions)
- [Credits](#credits)

<!-- tocstop -->

# Features

- Dynamic Updates: Automatically fetches and applies the latest LLM Bot user-agents rules from [ai-robots-txt](https://github.com/ai-robots-txt/ai.robots.txt), with support for additional sources
- Unified Bot Management: No longer maintain a robots.txt file for each of your applications
- Fast: Employs Aho-Corasick for efficient user-agent matching with intelligent caching
- Traffic Logging: Keep track of bot traffic for analysis and reporting
- Highly Configurable: Customize bot sources, HTTP responses, request routing, and defense strategies

# Usage

## Considerations

Please read if you plan to deploy this plugin!

- If providing a custom robots.txt template with `robotsTxtFilePath`, ensure that the `robots.txt` template is available to Traefik at startup. For Docker, this means passing the file in as a mount. For Kubernetes, mounting the template in a ConfigMap is easiest.
- The reverse proxy to the `botProxyUrl` is unbuffered. If you are passing this request to _another_ reverse proxy in front of a tarpit-style application, ensure proxy buffering is disabled.
- The cache from ai-robots-txt is refreshed if expired during a request. While this is set (by default) to update every 24 hours, there will be a small response speed impact (<0.06s) on the request that causes the cache refresh.

## Configuration

The follow parameters are exposed to configure this plugin

| Name | Default Value | Description |
|------|---------------|-------------|
|enabled|`true`|Whether or not the plugin should be enabled|
|cacheUpdateInterval|`24h`|How frequently sources should be refreshed for new bots. Also flushes the User-Agent cache.|
|cacheSize|`500`|The maximum size of the cache of User-Agent to Bot Name mappings. Rolls over when full.|
|botAction|`LOG`|How the bot should be wrangled. Available: `PASS` (do nothing), `LOG` (log bot info), `BLOCK` (log and return static error response), `PROXY` (log and proxy to `botProxyUrl`)|
|botProxyUrl|`""`|The URL to pass a bot's request to, if `PROXY` is the set `botAction`|
|botBlockHttpCode|`403`|The HTTP response code that should be returned when a `BLOCK` action is taken|
|botBlockHttpContent|`"Your user agent is associated with a large language model (LLM) and is blocked from accessing this resource"`|The value of the 'message' key in the JSON response when a `BLOCK` action is taken. If an empty string, the response body has no content.|
|logLevel|`INFO`|The log level for the plugin|
|robotsTxtFilePath|`""`| The file path to a custom robots.txt Golang template file. If one is not provided, a default will be generated based on the user agents from your `robotsSourceUrl`. See the `robots.txt` in the repo|
|robotsTxtDisallowAll|`false`|A config option to generate a robots.txt file that will disallow all user-agents. This does not change the blocking behavior of the middleware.|
|robotsSourceUrl|`https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt/robots.json`|A comma separated list of URLs to retrieve a bot list. You can provide your own, but read the notes below!|
|useFastMatch|`true`|When `true`, use an Aho-Corasick automaton for speedily matching uncached User-Agents against Bot Names. Consumes more memory. `false` relies on a slower, simple substring match.|

### Providing Custom Robots Sources

Presently, three different types of source files are supported.

- A rich JSON object, following the schema of the ai.robots.txt project's JSON file found [here](https://github.com/ai-robots-txt/ai.robots.txt/blob/main/robots.json).
  - This JSON format allows for additional metadata to be provided in logs which can be used for deeper analysis from administrators.
- Classic `robots.txt` styled formatting, from which a bots list will be extracted.
- Simple plain-text lists (newline separated) of bot names which should be blocked. Example [here](https://github.com/ai-robots-txt/ai.robots.txt/blob/main/haproxy-block-ai-bots.txt)

In any case, you should ensure that the server serving your source file provides a proper `Content-Type` header. Of particular note, using content from `raw.githubusercontent.com` **fails to do this**. If you wish to use a file hosted on GitHub, check out [jsdelivr](https://github.com/jsdelivr/jsdelivr?tab=readme-ov-file#github) which can proxy the file with the proper headers. It is recommended to pin the source to a specific git tag or commit.

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

2. (Optional) Create configMap for robots.txt template

If you want to use a custom `robots.txt` file template for the plugin to render, we'll need to create the configMap being referenced (ensure its in the same namespace!):

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
