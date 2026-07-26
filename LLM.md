# pulsard — canonical knowledge

`github.com/luxfi/pulsard` — offline threshold ML-DSA (FIPS-204) signer for the
Quasar **Pulsar** lane. Implements `warp.PulsarThresholdSigner`; emits
`warp.PulsarEvidence` (one standard ML-DSA-65 sig) verified by
`warp.VerifyPulsar`. Pinned: warp v1.23.0, crypto v1.19.17 (matches warp's pin
exactly → ONE crypto version in the build → sign/verify interop guaranteed).

## The hard truth
Threshold ML-DSA has NO native FIPS-204 construction (HighBits/r0 non-linearity
breaks FROST-additive). The dealerless path is research-grade TALUS
(arXiv:2603.22109). We DO NOT ship fake threshold math.

## What is real vs stub vs reference
- REAL: service shape (`signer.go`), FIPS-204 primitives (via `luxfi/crypto`),
  key-era records (reuse `warp.PulsarKeyEra`, never re-declared) + `KeyEraStore`
  resolver (`keyera.go`), session/nonce plumbing (`nonce.go`), typed round
  messages + phase state machine (`protocol.go`/`session.go`), mandatory release
  gate (`signer.go`: re-runs `warp.VerifyPulsar` before emitting).
- FAIL-CLOSED STUB: dealerless TALUS crypto. `ThresholdEngine` SPI (`engine.go`);
  default `Unimplemented()` returns `ErrThresholdMLDSAUnimplemented`. Design +
  engine contract in `docs/talus-design.tex` (LaTeX, per repo rule).
- REFERENCE / FOOTGUN: `ReferenceDealer` (`reference.go`) — single-party, holds
  the whole group secret, NOT threshold. Proves the verify integration only.

## Invariants
- `LaneContext = "LUX-QUASAR-PULSAR-MLDSA65-v1"` MUST equal warp's unexported
  `pulsarLaneContext`. We can't import it; the e2e test
  (`TestReferenceDealer_VerifiesUnderWarp`) is the drift guard — if it changes,
  the sig stops verifying and the test fails loudly.
- Subject MUST be 32 bytes (`ids.IDLen`); `ValidateSubject` fail-closes first.
- Reshare MUST preserve the group public key (advance Generation only); a
  key-changing reshare is rejected (`Signer.Reshare`).
- Suite is always `warp.SuitePulsarThresholdMLDSA65` ("Lux-Pulsar-TALUS-MLDSA65").

## Why luxfi/pulsar is NOT wired
`github.com/luxfi/pulsar` has a real circl-based TALUS engine BUT: keygen is
trusted-dealer, CSCP is semi-honest, and talus.go admits transcript
distinguishability. Wiring it as "dealerless" would overclaim. It is documented
as the candidate engine behind the SPI; registration is gated on the 5 contracts
in docs/talus-design.tex §7 (interop / dealerless keygen / malicious CSCP /
dual-PQ key independence / bounded distribution caveat).

## Build/test
`SDKROOT="$(xcrun --show-sdk-path)" GOWORK=off GOFLAGS=-mod=mod GOPRIVATE=github.com/luxfi/* go test ./...`
(23 tests, race-clean. The `ld: warning … luxcpp/install/lib` is a transitive
luxfi/accel C++ search-path warning via warp, not a pulsard error.)
