# vatdns - a toolset for vatsim to run proximity DNS/HTTP for connecting to FSD

## dnshaiku
dnshaiku processes DNS and HTTP requests for a configurable hostname, responding based upon request distance from FSD 
servers and network state.

## retardantfoam
retardantfoam is an external healthcheck for dnshaiku. If it fails passing healthchecks for a configurable amount of 
time, it pushes IP based lists to DigitalOcean Spaces and flushes content cache on Cloudflare. A human is required
to take action to return to using DNS/HTTP for FSD connections.

---
This toolset includes GeoLite2 data created by MaxMind, available from
<a href="https://www.maxmind.com">https://www.maxmind.com</a>.