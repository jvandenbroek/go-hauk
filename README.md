# go-hauk

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/sysHvT?referralCode=PNe-Vg)

Go port of the [Hauk](https://github.com/bilde2910/Hauk) location sharing backend.

## why

The original PHP implementation works fine but has some overhead. This port provides:

- 345x higher throughput (45k vs 133 req/s)
- 36x smaller image (16MB vs 578MB)
- 3x lower memory footprint (8MB vs 25MB idle)
- single static binary, no runtime dependencies
- redis only (no memcached)

## usage

run go-hauk and redis together with docker compose:

```
# generate a password hash
export HAUK_PASSWORD_HASH=$(htpasswd -nbBC 10 "" 'your-password' | tr -d ':\n')

docker compose up -d
```

or copy `.env.example` to `.env`, fill it in, and `docker compose up -d`.

### persistence

share data (sessions, links, locations) lives in redis with a per-share TTL
(`HAUK_MAX_DURATION`, default 24h). the bundled redis persists to a named volume
by default, so restarts and redeploys keep active shares alive. set
`REDIS_PERSIST=off` for in-memory only -- faster, but every share is lost on
restart.

### deploy on railway

use the **Deploy on Railway** button above.

## config

all config via environment variables:

| var | default | description |
|-----|---------|-------------|
| HAUK_LISTEN_ADDR | :8080 | listen address |
| HAUK_PUBLIC_URL | http://localhost:8080/ | public url for share links |
| HAUK_REDIS_ADDR | localhost:6379 | redis address (host:port or redis:// url) |
| HAUK_AUTH_METHOD | password | auth method (password, htpasswd, ldap) |
| HAUK_PASSWORD_HASH | | bcrypt hash for password auth |
| HAUK_MAX_DURATION | 86400 | max share lifetime in seconds (redis key TTL) |
| HAUK_RATE_LIMIT_AUTH | 10 | max auth requests per minute per ip |
| HAUK_RATE_LIMIT_ADOPT | 10 | max adopt requests per minute per ip |
| HAUK_TRUST_PROXY | true | trust X-Forwarded-For (set false if not behind proxy) |
| HAUK_SPEED_FONT_SIZE | 1.6em | size of the speed on the map markers (css size: px, em, rem, pt, %) |
| HAUK_AUTO_ZOOM | true | speed-dependent zoom in the viewer while following a user |
| HAUK_AUTO_ZOOM_STEPS | 0:17,10:16,30:15,60:14,90:13,120:12 | `speed:zoom` breakpoints, speed in km/h (regardless of HAUK_VELOCITY_UNIT) |
| HAUK_AUTO_ZOOM_SMOOTHING | 0.3 | weight of the newest speed in the moving average, (0, 1], 1 = no smoothing |
| HAUK_AUTO_ZOOM_MIN_INTERVAL | 3 | min seconds before zooming in after an automatic zoom change |
| HAUK_AUTO_ZOOM_OUT_INTERVAL | 5, or MIN_INTERVAL if lower | min seconds before zooming out after an automatic zoom change |
| HAUK_AUTO_ZOOM_JUMP_THRESHOLD | 2 | zoom levels from the target at which it zooms straight there instead of one level per change, 0 = always one level |
| HAUK_AUTO_ZOOM_MAX_ACCURACY | 30 | ignore speeds of points with a worse GPS accuracy (m), 0 = off; points older than 3 share intervals (min 10 s) are ignored too |
| HAUK_AUTO_ZOOM_HYSTERESIS | 5 | km/h the average speed must exceed a breakpoint by before zooming out (zooming in happens as soon as it drops below) |
| HAUK_AUTO_ZOOM_RESUME | 300 | seconds after a manual zoom before auto zoom resumes, 0 = never |
| HAUK_LOOKAHEAD | 0 | while following, shift the user against the direction of travel by this fraction of the map size (e.g. 0.25) to show more of the road ahead, 0 = off, max 0.45 |

see `config/config.go` for full list.

### auto zoom

while following a user, the viewer zooms in at low speed and out at high speed.
manually zooming (wheel, pinch, double click, +/- buttons, keyboard) pauses it;
the `AUTO` map button shows the state (filled = on, outlined = paused, struck
through = off) and toggles it; while paused, a label next to it counts down to
the resume. switching it off is remembered in the browser. `HAUK_AUTO_ZOOM=false`
only makes it off by default, viewers can still turn it on.

viewers can override the server defaults in the share link, after the share
id or in the hash:

| parameter | example | effect |
|-----------|---------|--------|
| autozoom | `?ABCD-1234&autozoom=0` | auto zoom off (`1` = on), takes precedence over the remembered choice |
| zoom | `?ABCD-1234&zoom=16` | fixed zoom level while following, disables auto zoom |
| lookahead | `?ABCD-1234&lookahead=0.25` | look-ahead fraction, `0` = off |

e.g. `https://hauk.example.com/?ABCD-1234#autozoom=0&lookahead=0.2`.

## security improvements over upstream

- adopt authorization: only share owner can adopt into groups (fixes CVE-like auth bypass in upstream)
- built-in rate limiting on auth and adopt endpoints (configurable, default 10 req/min/ip)

## disclaimer

this is a best-effort port of the upstream reference backend. the security
surface has been mitigated where practical, but no third-party audit has been
performed, and risks inherent in the upstream protocol (e.g. 6-digit group pins,
view links readable by anyone with the url) are carried forward. use at your own
discretion.

some proposed hardening lives under https://github.com/parkan/go-hauk/pull/10/changes
however validating this not breaking things without conformance tests is nontrivial

## compatibility

drop-in replacement for the php backend. works with the existing android app and web frontend.

## license

same as upstream (apache 2.0).
