# Xray Core Project Review

**Review date:** 2026-07-19  
**Repository:** `/workspaces/alice`  
**Review mode:** Read-only; no source files were modified.

## Executive summary

The project has a broad and mature Go architecture, but the review identified several high-impact security, reliability, concurrency, and lifecycle issues. The most urgent items are:

1. A Shadowsocks authentication bypass involving mixed `NONE` and AEAD users.
2. Unauthenticated exposure of pprof and expvar through the metrics service.
3. World-readable TLS/REALITY key-log files.
4. Confirmed data races in XHTTP and TLS OCSP refresh paths, plus a mux/XUDP migration race.
5. Multiple configuration and malformed-input paths that panic instead of returning errors.
6. UDP dispatcher ownership leaks, silent datagram truncation, and incomplete shutdown.
7. uTLS connections silently losing configured TLS security and protocol settings.
8. Missing PR/push CI validation and nondeterministic or time-sensitive test/build inputs.

The recommended order is to fix authentication and diagnostic exposure first, then address concurrency and resource ownership, configuration validation, routing/handler lifecycle, and CI/reproducibility.

## Scope and architecture

The review covered the principal runtime and configuration areas:

- `app`: metrics, DNS, observatory, routing, proxyman, reverse portal, and statistics.
- `common`: buffers, networking, crypto, mux, matching, retry, and protocol utilities.
- `core`: instance creation, configuration loading, and feature lifecycle.
- `infra`: configuration parsing and code-generation tooling.
- `main`: CLI and external configuration loading.
- `proxy`: VLESS, Shadowsocks, DNS, and related protocol implementations.
- `transport`: TLS, REALITY, XHTTP/split HTTP, UDP, TCP, and dialers.

The repository contains approximately 605 Go files and 127 test files. The primary entry points are `main/main.go`, `core/xray.go`, `core/config.go`, and `main/run.go`.

## Findings

### 1. Shadowsocks `NONE` can bypass authentication — High

**Location:** `proxy/shadowsocks/validator.go:28-36,111-150`

A non-AEAD user is rejected only when another user already exists. Therefore, a `NONE` user can be added first, followed by AEAD users. During lookup, the first non-AEAD user is returned without packet authentication:

```go
} else {
    u = user
    ivLen = user.Account.(*MemoryAccount).Cipher.IVSize()
    return
}
```

Unauthenticated traffic can consequently be attributed to that account. Mixed `NONE`/AEAD configurations must be rejected regardless of insertion order, or every candidate must be validated before selection.

### 2. Metrics exposes unauthenticated pprof and expvar — High

**Location:** `app/metrics/metrics.go:7,41-65,92-119`

The package imports `net/http/pprof` and serves `http.DefaultServeMux`. A configured non-loopback listener therefore exposes diagnostic endpoints such as `/debug/pprof` and `/debug/vars` without authentication.

Use a dedicated HTTP mux, bind diagnostics to loopback by default, and require explicit authentication for remote access.

### 3. Metrics handlers can panic on duplicate expvar registration — High

**Location:** `app/metrics/metrics.go:41-84`

Every `NewMetricsHandler` calls `expvar.Publish("stats", ...)` and `expvar.Publish("observatory", ...)`. Constructing a second handler in the same process causes duplicate global expvar registration and can panic, affecting reloads or repeated setup.

Register these variables once, use unique names, or avoid global expvar registration in favor of a handler-local metrics mux.

### 4. Metrics counter rendering can panic on arbitrary counter names — Medium

**Location:** `app/metrics/metrics.go:51-53`, `app/stats/stats.go:37-49`

`RegisterCounter` accepts arbitrary names, but metrics rendering assumes at least four `>>>`-separated components:

```go
nameSplit := strings.Split(name, ">>>")
typeName, tagOrUser, direction := nameSplit[0], nameSplit[1], nameSplit[3]
```

A short name such as `x` causes an index-out-of-range panic while rendering `/debug/vars`. Validate the name format or render malformed names safely.

### 5. Metrics listeners are not closed — Medium/High

**Location:** `app/metrics/metrics.go:92-107,131-133`

`Start` creates a direct TCP listener and an outbound listener, but `Close` is a no-op. Shutdown or reload can leave listeners, goroutines, and ports active.

Track both servers/listeners and close them explicitly. This also prevents stale diagnostics endpoints after a configuration reload.

### 6. TLS and REALITY key logs are world-readable — High

**Location:** `transport/internet/tls/config.go:461-467`, `transport/internet/reality/config.go:61-71`

Key-log files are created with mode `0644`. TLS key logs can contain session secrets that allow captured traffic to be decrypted. They should normally be created with mode `0600`, with ownership and lifecycle managed explicitly.

### 7. uTLS silently drops configured TLS settings — High

**Location:** `transport/internet/tls/tls.go:124-143`

`UClient` and `GeneraticUClient` pass a manually reduced copy of `tls.Config` to uTLS. `copyConfig` omits settings including:

- `NextProtos`
- `MinVersion` and `MaxVersion`
- `CipherSuites`
- `CurvePreferences`
- `SessionTicketsDisabled`
- other certificate/client-auth and handshake options

Callers such as `transport/internet/tcp/dialer.go:67-76` and `transport/internet/splithttp/dialer.go:117-123` can therefore negotiate settings different from the configured policy. Replace the incomplete manual copy with a maintained conversion that preserves all supported security-relevant fields, and test custom fields through a uTLS handshake.

### 8. Remote configuration bodies are unbounded — High

**Location:** `main/confloader/external.go:52-74`, `common/buf/multi_buffer.go:11-23,78-95`

External HTTP configuration responses are read until EOF without a maximum size. A malicious or compromised endpoint can cause excessive memory consumption during startup.

Use an `io.LimitReader` or equivalent bounded reader before parsing, and return a clear size-limit error.

### 9. Confirmed XHTTP and TLS OCSP races — High

**Locations:**

- `transport/internet/splithttp/client.go:262-297`
- `transport/internet/splithttp/upload_queue.go:59-103`
- `transport/internet/splithttp/dialer.go:446-502`
- `transport/internet/tls/config.go:90,127,253`

XHTTP read/set/close paths access shared readers and queue state concurrently. Asynchronous upload processing can also observe pooled buffers after they are cleared or reused. TLS OCSP refresh writes certificate state while handshakes read it.

The race detector reproduced races in XHTTP and TLS OCSP paths. Synchronize shared state and define buffer ownership before starting asynchronous work.

### 10. XUDP migration updates shared state outside its lock — High

**Locations:** `common/mux/server.go:241-255`, `common/mux/session.go:225-227,241-245`

XUDP migration assigns `x.Mux` and `x.Status` after releasing the manager lock, while expiry and interruption paths read those fields under the lock. Concurrent migration and expiry can observe partially initialized or stale state and may race on the mux pointer.

Perform the complete state transition under the same lock, or encapsulate it in an atomic/session-level method. Add a race-enabled migration/expiry test.

### 11. UDP dispatcher leaks buffers and truncates datagrams — High

**Location:** `transport/internet/udp/dispatcher.go:108-125,200-228`

Problems include:

- `Dispatch` does not release its owned payload when ray creation fails.
- `Dispatch` does not release its payload when `conn.link.Writer` is nil.
- `ReadFrom` copies a received payload but never releases the pooled buffer.
- `WriteTo` allocates a fixed 8192-byte buffer and silently truncates larger datagrams while returning a nil error.

Make ownership explicit on every path, release payloads after reads, and allocate according to datagram size or return an explicit oversize error.

### 12. Closing the UDP dispatcher leaves routed resources active — High

**Location:** `transport/internet/udp/dispatcher.go:172-185,231-233`

`dispatcherConn.Close` only closes `c.done`; it does not call `Dispatcher.RemoveRay` or cancel the active routed connection. Background handlers and links can remain active until inactivity timeout.

Make close idempotently remove the ray, cancel its context, interrupt both link sides, and drain/release queued packets.

### 13. Configuration decoder lookup can call a nil function — High

**Location:** `infra/conf/serial/builder.go:31-36,57-63`

Only JSON is registered in `ReaderDecoderByFormat`, while YAML, TOML, JSONC, and YML are advertised elsewhere. An unregistered format returns a nil decoder and panics when called.

Return an unsupported-format error or register the advertised decoders.

### 14. Multiple malformed configuration inputs panic — High

Relevant locations include:

- Empty listen domain: `infra/conf/xray.go:141`
- Nil FakeDNS address: `infra/conf/fakedns.go:77-80`
- Malformed REALITY SpiderX: `infra/conf/transport_internet.go:731-739`
- Missing SOCKS server address: `infra/conf/socks.go:98-104`
- Null VLESS fallback/vnext entries: `infra/conf/vless.go:147-163,239-242`
- Null Shadowsocks users: `infra/conf/shadowsocks.go:61-65,118-135`

These paths should validate nil, empty, and malformed values and return configuration errors instead of dereferencing or indexing blindly.

### 15. VLESS UUID normalization causes account collisions — High

**Location:** `proxy/vless/validator.go:21-58`

`ProcessUUID` clears bytes 6 and 7 of every UUID. Distinct UUIDs that differ only in those bytes map to the same key. The later account overwrites the earlier one, and deleting the earlier account can delete the newer mapping.

Reject normalized-ID collisions and make deletion conditional on the mapping still referring to the account being removed.

### 16. Duplicate mux session IDs overwrite active sessions — Medium/High

**Location:** `common/mux/session.go:75-85`

Adding a duplicate session ID replaces the existing session without rejecting or closing it. The old session can retain goroutines and resources while new frames are routed elsewhere.

Reject duplicates or close the previous session before replacement.

### 17. Router reload is not transactional — High

**Location:** `app/router/router.go:107-152`

Replacement reloads clear existing rules and balancers before the new configuration is fully validated. A later error leaves the router empty or partially rebuilt.

Build a complete replacement under temporary state and swap it only after successful validation.

### 18. Router rule access races with reloads — High

**Location:** `app/router/router.go:199-228`

`pickRouteInternal` iterates over `r.rules` without the mutex while reload and removal mutate the same slice under lock. This can race and observe inconsistent routing state.

Use an immutable snapshot or hold the appropriate read lock for the full rule-selection operation.

### 19. Invalid routing regexes panic — High

**Location:** `app/router/config.go:70-75`

User-provided routing attributes are compiled with `regexp.MustCompile`. Invalid configuration therefore terminates the process instead of returning an error.

Use `regexp.Compile` and propagate the error.

### 20. Handler removal and failed startup leak resources — High

**Locations:**

- `app/proxyman/outbound/outbound.go:103-145`
- `app/proxyman/inbound/inbound.go:38-57`
- `app/reverse/portal.go:56-65`

Handlers are registered before `Start`, and failed starts leave broken entries registered. Removal deletes map entries without calling `Close`, leaving listeners and workers active. Portal close inherits the same problem.

Start before publication or roll back failed registration, and close handlers before removing them.

### 21. UDP worker shutdown does not close active connections — High

**Location:** `app/proxyman/inbound/worker.go:439-465`

`udpWorker.Close` closes the hub and periodic checker but does not close active per-connection entries. Existing goroutines can remain blocked after shutdown.

Track and close all active connections before returning from `Close`.

### 22. Maximum-width inbound port ranges can loop forever — High

**Location:** `app/proxyman/inbound/always.go:128-166`

An unsigned loop that increments through `math.MaxUint32` wraps to zero and never terminates. Reject ranges ending at the maximum value or use a terminating loop structure.

### 23. Observatory returns internal mutable state — High

**Location:** `app/observatory/observer.go:40-41`

`GetObservation` returns the internal status slice without locking or copying. Background probes update the same objects under a mutex, while callers can mutate them directly.

Return a synchronized deep snapshot.

### 24. Invalid observatory URLs can panic — High

**Location:** `app/observatory/observer.go:160-168`

The error from `http.NewRequest` is ignored. A malformed configured URL leaves `req` nil before `req.Header.Set` is called.

Handle the request-construction error before accessing the request.

### 25. Health-ping result access panics before the first sample — Medium

**Location:** `app/observatory/burst/healthping_result.go:44-55`

`GetWithCache` indexes `h.rtts[h.idx]` while `rtts` is still nil. Return empty statistics until a first sample exists.

### 26. Health-ping interval minimum is compared in nanoseconds — Medium

**Location:** `app/observatory/burst/healthping.go:65-70`

`settings.Interval < 10` compares a `time.Duration` to ten nanoseconds, not ten seconds. Most invalid sub-10-second values bypass the intended minimum.

Compare against `10 * time.Second`.

### 27. Fake DNS close races with lookups — High

**Location:** `app/dns/fakedns/fake.go:51-55`

`Close` sets internal maps, IP range, and mutex pointers to nil without synchronizing with readers. Concurrent or post-close calls can race or dereference nil pointers.

Introduce a closed state and synchronize shutdown with all lookup methods.

### 28. Static host construction dereferences nil and mutates input — High

**Location:** `app/dns/hosts.go:20-31`

A nil host mapping panics. Additionally, the constructor sets each caller-provided slice entry to nil, destroying reusable configuration data.

Validate entries and never mutate the caller's slice.

### 29. QUIC DNS request buffers leak — Medium

**Location:** `app/dns/nameserver_quic.go:119-159`

`dnsReqBuf` is allocated from the buffer pool but is not released on success or error paths.

Release it with a defer immediately after allocation.

### 30. QUIC DNS connection creation has a check-then-act race — Medium

**Location:** `app/dns/nameserver_quic.go:214-245`

Multiple callers can observe an unavailable connection, then each acquire the write lock and open a new connection without rechecking. Earlier connections can be orphaned or overwritten.

Recheck under the write lock or use a singleflight/once-style connection initializer.

### 31. Online-map cleanup and map exposure race — Medium/High

**Location:** `app/stats/online_map.go:38-52,78-85`

`lastCleanup` is accessed outside the mutex, and `IpTimeMap` returns the internal mutable map directly. Callers can cause concurrent map access and data races.

Protect cleanup state and return a copy of the map.

### 32. Channel close can race with deferred delivery — High

**Location:** `app/stats/channel.go:152-172`

When a subscriber is full, a goroutine retries delivery. Closing the subscriber channel before the retry executes can cause `send on closed channel`.

Coordinate publisher shutdown and deferred sends, or avoid closing subscriber channels until all publishers have stopped.

### 33. Empty SRV responses panic — Medium

**Location:** `transport/internet/dialer.go:185-192`

A valid successful SRV lookup can return zero records. The unconditional `srvRecords[0]` access panics.

Check the record count and return no override or a controlled error.

### 34. Empty Unix socket names panic — Medium

**Location:** `transport/internet/system_listener.go:114-120`

An empty `net.UnixAddr.Name` reaches `address[0]`. Validate the name before indexing.

### 35. ECH UDP IPv6 endpoints omit the default port — Medium

**Location:** `transport/internet/tls/ech.go:276-283`

The code appends `:53` only when the string contains no colon. `udp://[2001:db8::1]` contains colons but has no port, so parsing fails.

Use structured host/port parsing that distinguishes an IPv6 literal from an explicitly supplied port.

### 36. VLESS large writes report the wrong byte count — Medium

**Location:** `proxy/vless/encryption/common.go:47-77`

Writes larger than one encryption chunk return the length of only the final encrypted chunk instead of the total plaintext bytes accepted.

Accumulate and return the total written count, or return an error if only partial input was consumed.

### 37. DNS blocked queries can terminate the entire TCP session — Medium

**Location:** `proxy/dns/dns.go:179-207`

A blocked query can cause the session-level handler to exit instead of isolating the failed query.

Handle per-query cancellation independently from the TCP connection lifecycle.

### 38. Several Shadowsocks 2022 error paths leak connections — Medium

**Location:** `proxy/shadowsocks_2022/outbound.go:89-157`

Connections created before later validation or setup errors are not consistently closed. Add deferred cleanup until ownership is transferred.

### 39. Formatter subprocesses are not synchronized — Medium

**Location:** `infra/vformat/main.go:93-109,190-193`

Formatter goroutines are launched without waiting for completion or reliably propagating errors. Multiple passes can overlap, and the parent process can exit while child work is pending.

Use an `errgroup`/wait group and return subprocess errors.

### 40. Configuration loader registration is not rolled back — Medium

**Location:** `core/config.go:44-60`

`RegisterConfigLoader` updates the name map and earlier extension entries before discovering a duplicate later extension. A failed registration leaves ghost state.

Validate all names/extensions first, then commit the maps atomically.

## Validation performed

The following commands were run during the review:

```text
git status --short --branch
git diff --stat
git log -1 --oneline
git diff --check
go test ./...
go vet ./...
go test -race ./transport/internet/splithttp ./proxy/dns ./proxy/vless/encryption
```

### Test results

`go test ./...` currently fails in:

- `app/router`: missing `geoip.dat`
- `common/reflect`: `TestMarshalConfigJson`
- `infra/conf`: `unknown config id: vmess`

`go vet ./...` reports:

- discarded cancellation functions
- protobuf-generated `MessageState` values copied after containing mutexes

Race testing reports races in:

- XHTTP/split HTTP paths
- TLS OCSP refresh paths

## CI and reproducibility

**Location:** `.github/workflows/build.yaml:3-25`

The workflow runs only for published releases and does not provide a normal push/PR gate for tests, vet, race detection, or builds. It also uses a floating Go version range.

Additional issues:

- `.goreleaser.yaml:7-10` uses `GOEXPERIMENT=jsonv2`.
- `README.md:176-193` documents build commands without the same experiment flag.
- Some tests use live external network services, including `transport/internet/tls/ech_test.go:13-39` and `app/dns/nameserver_doh_test.go:16-31`.
- Test assets such as `geoip.dat` are not consistently available in the test environment.

Recommended CI changes:

1. Add deterministic push/PR workflows for build, test, vet, and selected race targets.
2. Separate live integration tests from hermetic unit tests.
3. Pin the exact Go toolchain used for releases.
4. Align README, CI, and GoReleaser build flags.
5. Make required test assets explicit and reproducibly provisioned.

## Working-tree observations

At review time, the working tree contained changes to:

```text
common/protocol/tls/cert/ca.crt
common/protocol/tls/cert/ca.key
```

The certificate and key matched cryptographically. The certificate metadata was:

- Subject: `O=Xray Inc, CN=Xray Root CA`
- Not before: `2026-07-19 08:02:35 UTC`
- Not after: `2026-07-19 12:02:35 UTC`

These appear to be short-lived test fixtures, but their expiry makes tests time-sensitive. No changes were made to them.

## Recommended remediation order

1. Fix the Shadowsocks `NONE` authentication bypass.
2. Restrict metrics/pprof exposure and secure key-log permissions.
3. Fix uTLS configuration copying so configured security settings are preserved.
4. Resolve the confirmed XHTTP, TLS OCSP, mux migration, router, observatory, and online-map races.
5. Fix UDP buffer ownership, oversized datagram handling, and dispatcher shutdown.
6. Make configuration parsing and routing validation return errors instead of panicking.
7. Make handler removal, failed startup, portal shutdown, and UDP worker shutdown fully close resources.
8. Fix VLESS UUID collision handling and mux duplicate-session behavior.
9. Add regression tests for all panic and ownership paths.
10. Add deterministic PR/push CI and align toolchain/build settings.

## Review status

This report records findings only. No fixes, commits, or pull requests were made.
