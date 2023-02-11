# vatdns - a toolset for vatsim to run proximity DNS for connecting to FSD

## tl;dr

### dnshaiku
dnshaiku handles DNS lookup requests for a configurable hostname. Possible IP addresses to respond to a request with are
populated by dataprocessor. dnshaiku only considers servers that are accepting connections (see dataprocessor). dnshaiku
handles checking distance to servers and serving the lowest populated server based upon the country of the server selected
by distance. This is to cover cases where we have multiple FSD servers in one location (eg: USA-EAST or GERMANY). We want
the one with least connections.

### retardantfoam
retardantfoam is an external healthcheck for dnshaiku and dataprocessor. If it fails passing healthchecks for a configurable
amount of time, it pushes IP based lists to DigitalOcean Spaces and flushes content cache on Cloudflare. A human is required
to take action to return to using DNS for FSD connections.
