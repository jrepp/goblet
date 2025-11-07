# Goblet: Git caching proxy

Goblet is a Git proxy server that caches repositories for read access. Git
clients can configure their repositories to use this as an HTTP proxy server,
and this proxy server serves git-fetch requests if it can be served from the
local cache.

In the Git protocol, the server creates a pack-file dynamically based on the
objects that the clients have. Because of this, caching Git protocol response
is hard as different client needs a different response. Goblet parses the
content of the HTTP POST requests and tells if the request can be served from
the local cache.

This is developed to reduce the automation traffic to googlesource.com. Goblet
would be useful if you need to run a Git read-only mirroring server to offload
the traffic.

This is not an official Google product (i.e. a 20% project).

## Usage

Goblet is intended to be used as a library. You would need to write some glue
code. This repository includes the glue code for googlesource.com. See
`goblet-server` and `google` directories.

## Offline Mode and Resilience

Goblet can now serve ls-refs requests from the local cache when the upstream server is unavailable:

- **Automatic fallback**: When upstream is down, Goblet serves cached ref listings from the local git repository
- **Graceful degradation**: Git operations continue to work with cached data during upstream outages
- **Staleness tracking**: Logs warnings when serving refs older than 5 minutes
- **Testing support**: Upstream can be disabled for integration testing

### Configuration

By default, Goblet attempts to contact upstream servers and falls back to local cache on failure. For testing scenarios where you want to disable upstream connectivity entirely:

```go
falseValue := false
config := &goblet.ServerConfig{
    LocalDiskCacheRoot: "/path/to/cache",
    // ... other config ...
    UpstreamEnabled: &falseValue,  // Disable all upstream calls (testing only)
}
```

When `UpstreamEnabled` is `nil` or points to `true` (default), Goblet operates in production mode with automatic fallback to local cache on upstream failures.

## Limitations

While Goblet can serve ls-refs from cache during upstream outages, fetch operations for objects not already in the cache will still fail if the upstream is unavailable. This is expected behavior as Goblet cannot serve content it doesn't have cached.
