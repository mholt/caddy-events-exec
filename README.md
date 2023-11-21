Run commands on Caddy events
============================

This is the `events.handlers.exec` Caddy module. It executes commands to handle Caddy events.

**It is EXPERIMENTAL and subject to change.** After getting some production experience, if demand is high enough and if this is generally useful to most users, we may move this into the standard Caddy distribution. It would be the only Caddy module that executes commands on the system, so we want to make sure it does not allow for _arbitrary_ commands to be executed.

> [!NOTE]
> This is not an official repository of the [Caddy Web Server](https://github.com/caddyserver) organization.

## Install

Like any other Caddy plugin, [select it on the download page](https://caddyserver.com/download) to get a custom build, or use xcaddy to build from source:

```
$ xcaddy build --with github.com/mholt/caddy-events-exec
```

## Usage

Minimal JSON config example:

```json
{
	"apps": {
		"events": {
			"subscriptions": [
				{
					"events": ["cert_obtained"],
					"handlers": [
						{
							"handler": "exec",
							"command": "systemctl",
							"args": ["reload", "mydaemon"]
						}
					]
				}
			]
		}
	}
}
```

This will run `systemctl reload mydaemon` every time Caddy obtains a certificate. Of course, you will need to make sure `caddy` has permission to run any command you configure it with.

Equivalent Caddyfile:

```
{
	events {
		on cert_obtained exec systemctl reload mydaemon
	}
}
```

## Notes

This module runs commands in the _background_ by default, to avoid performance problems caused by blocking.

If you want the ability to cancel an event (for example, `cert_obtaining` can be canceled to avoid getting a certificate) you must configure the command to run in the foreground as well as the exit code(s) to trigger an abort.

Please be mindful of any security implications of the commands you run and how you configure this module, at least until we get more production experience. Please test it out and report any issues!
