
+++
date = "2017-11-11T17:30:53-08:00"
title = "version"
repo = "version"
url = "/version"
+++

# version

The version package offers simple helper utilties for tracking and exposing the version information associated with your Go binaries.

## Tracking

We use [the link tool](https://golang.org/cmd/link/)'s `-X` option to embed values at build time. The values are:

* `Hash`: the VCS commit hash the binary was built from
* `Branch`: the VCS branch the binary was built from
* `Tag`: the VCS tag the binary was built from
* `Timestamp`: the date and time the binary was built, in ISO 8601 format.

To populate these values, you need to pass the `-ldflags` command line argument, with the `-X` option used to specify variables:

```
go build -ldflags "-X darlinggo.co/version.Hash={YOUR_HASH_HERE} -X darlinggo.co/version.Tag={YOUR_TAG_HERE} -X darlinggo.co/version.Branch={YOUR_BRANCH_HERE} -X darlinggo.co/version.Timestamp={YOUR_TIMESTAMP_HERE}" .
```

Obviously, that gets annoying to type out every time you build, so we made it a bit easier with the included `ldflags.sh` script, which exports the $LDFLAGS environment variable when it's called, populated wtih all the right information. You can use this on the commandline:

```
$GOPATH/src/darlinggo.co/version/ldflags.sh
go build -ldflags ${LDFLAGS} .
```

Or you can include it in your own shell script:

```sh
source $GOPATH/src/darlinggo.co/version/ldflags.sh
go build -ldflags ${LDFLAGS} .
```

### Using with vendoring

If you vendor this dependency (and you should!) you need to tell the `ldflags.sh` script where to look, unfortunately, because your package's import path becomes part of its import path. Fortunately, you can do this with the `PACKAGE_PREFIX` variable:

```sh
export PACKAGE_PREFIX=my/package/import # e.g., darlinggo.co/version
source vendor/darlinggo.co/version/ldflags.sh
go build -ldflags "${LDFLAGS}" .
```

Note that we'll automatically include the vendor part if you have a `PACKAGE_PREFIX` set, so you don't have to include that.

### Customising for non-Git versioning

Right now, the values supplied by `ldflags.sh` all rely on the package being tracked with Git to work. While supporting other VCSes is possible, and would be nice, it's not really a priority at the moment—which is a non-subtle way of saying “pull requests accepted” 😉. However, you can supply your own values to override any or all of the Git values, using environment variables:

* `VERSION` will override the Git hash.
* `TAG` will override the Git tag.
* `BRANCH` will override the Git branch.
* `TIMESTAMP` will override the `date`-command generated timestamp.

## Exposing

The version information is stored in exported variables, so you can expose it however you want:

* `version.Hash` has the hash.
* `version.Tag` has the tag.
* `version.Branch` has the branch.
* `version.Timestamp` has the timestamp.

Version also supplies its own `http.Handler` that writes the version information to the Response. Just associate it with a server:

```go
http.Handle("/version", version.Handler)
err := http.ListenAndServe(":8080", nil)
if err != nil {
	panic(err)
}
```

The version will be, by default, output in the following format:

```
VERSION_HASH="{YOUR HASH HERE}"
VERSION_TAG="{YOUR TAG HERE}"
VERSION_BRANCH="{YOUR BRANCH HERE}"
VERSION_TIME="{YOUR TIMESTAMP HERE}"
```

If the Accept header includes "application/json", however, the output will be in the following format:

```json
{
	"hash": "{YOUR HASH HERE}",
	"tag": "{YOUR TAG HERE}",
	"branch": "{YOUR BRANCH HERE}",
	"timestamp": "{YOUR TIMESTAMP HERE}"
}
```

