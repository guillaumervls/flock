# Flock

Deploy multiple Fly.io applications (defined in their `fly.toml`s) together.

# Install

```sh
# TODO: Provide install instructions
# For now (having Go installed):
git clone https://github.com/guillaumervls/flock.git
cd flock
go run main.go up # <- deploy what's in the "example" directory
```

# Usage

> Get additional help by running `flock help`.

You can select the Fly.io organization you want to deploy in with the `--org` flag
(defaults to your personal organization).

## up

Find `fly.toml` and `*.fly.toml` files recursively from the current directory,
and deploy them:

```sh
flock up
```

The `**/fly.toml,**/*.fly.toml` glob patterns are the default.
Change that with `--flytoml-glob` argument(s).
For multiple patterns pass them with mutiple `--flytoml-glob` or comma-separated in one go.

### Variables

In any of these `fly.toml` files you can insert environment
variables with the `$VAR` (or `${VAR}`) syntax.

If needed variables are not set, it will ask you.

It will also read environment variables from an `.env` file.
Change that with the `--env-file` flag.
For multiple files pass them with mutiple `--env-file` flags or comma-separated in one go.
In case of duplicate variables, the _first_ file has precedence.
And _actual_ environments variables have ultimate precedence.
It will also save to this file (to the _first_ one if you passed several)
the variables you've input (so you don't need to re-input these later).

### App dependencies

Add the following section to your `fly.toml` / `*.fly.toml` files,
to specify dependency on other apps / secrets:

```toml
[[flock.dependencies.apps]] # wait for the deployment of other apps
name = "other-app"

[[flock.dependencies.secrets]]
name = "SECRET_KEY"
source.app = "other-app" # will prompt for the secret if non existent
source.name = "SECRET_NAME_IN_OTHER_APP" # defaults to the same name
```

> **NB:** adding a dependency on another app's secret doesn't necessarily wait
> for that app to be deployed. For this behavior, add it in `flock.dependencies.apps`.

## down

Destroy every app deployed by `flock up`:

```sh
flock down
```
