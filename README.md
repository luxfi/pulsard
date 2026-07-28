# pulsard

The offline threshold ML-DSA (FIPS-204) signer for the Lux Quasar **Pulsar**
finality lane. It implements `warp.PulsarThresholdSigner`: a `Signer` turns a
32-byte subject into one `warp.PulsarEvidence` carrying a single, standard
FIPS-204 ML-DSA-65 signature that the chain verifies with `warp.VerifyPulsar`.

```
PRODUCE (offline, here)                  VERIFY (on chain, in warp)
  dealerless nonce DKG, BCC, CEF,  ──►    one FIPS-204 ML-DSA-65 verify
  CSCP, blame, aggregate (TALUS)         (warp.VerifyPulsar)
```

## Honest status

Threshold ML-DSA has **no native FIPS-204 construction**; the dealerless path is
the research-grade TALUS construction (arXiv:2603.22109). This module is
deliberately split into what is sound and what is pending:

| Part | Status | Where |
|------|--------|-------|
| Service shape (`warp.PulsarThresholdSigner`) | **REAL** | `signer.go` |
| FIPS-204 ML-DSA-65 primitives (`luxfi/crypto`, same lib warp verifies with) | **REAL** | via `keyera.go`, `reference.go` |
| Key-era / group-key records (reuse `warp.PulsarKeyEra`) + resolver | **REAL** | `keyera.go` |
| Session/nonce plumbing (session id, non-grindable nonce pool) | **REAL** | `nonce.go`, `session.go` |
| Typed protocol messages + phase state machine | **REAL** | `protocol.go`, `session.go` |
| Mandatory release gate (re-verify via `warp.VerifyPulsar`) | **REAL** | `signer.go` |
| **Dealerless TALUS threshold crypto** | **FAIL-CLOSED STUB** | `engine.go` (`ErrThresholdMLDSAUnimplemented`), design in `docs/talus-design.tex` |
| Trusted-dealer single-party engine | **REFERENCE / DEV-TEST FOOTGUN** | `reference.go` |

No fake threshold math is shipped. The default engine fails closed; the only
engine that produces a signature is the clearly-labelled, non-threshold
`ReferenceDealer`, which exists solely to prove the verify integration.

## The boundary guarantee

`Signer.ThresholdSign` never returns evidence that does not verify: whatever
engine is plugged in, the signature is re-checked with `warp.VerifyPulsar`
(the chain's own verifier) before it leaves the process. Emitted-evidence
correctness is a property of pulsard, independent of the engine.

## Build & test

```sh
SDKROOT="$(xcrun --show-sdk-path)" GOWORK=off GOFLAGS=-mod=mod GOPRIVATE=github.com/lux-private/* \
  go build ./... && go test ./...
```

## Demo (dev/test footgun)

```sh
go run ./cmd/pulsard -mode reference   # produces a real group sig, VERIFIED via warp.VerifyPulsar
go run ./cmd/pulsard -mode threshold   # FAIL-CLOSED: ErrThresholdMLDSAUnimplemented (as designed)
```

## Wiring a real engine

A concrete dealerless engine (candidate: `github.com/luxfi/pulsar`) may be
registered behind the `ThresholdEngine` SPI **only** after the interop and trust
contracts in `docs/talus-design.tex` (§7) are established and reviewed:
interop under `mldsa65.Verify`, dealerless keygen, malicious-secure CSCP,
dual-PQ key independence, and a bounded transcript-distribution caveat.
