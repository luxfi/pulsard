// Copyright (C) 2019-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Command pulsard is the thin service shell around the pulsard library. It has
// two modes:
//
//	pulsard threshold   (default) — the real dealerless signer. It is FAIL-CLOSED
//	                                 until a sound ThresholdEngine is wired, so it
//	                                 exits non-zero with ErrThresholdMLDSAUnimplemented.
//	pulsard reference   — DEV/TEST FOOTGUN. A single-party trusted dealer signs a
//	                      subject and self-verifies the evidence through the chain's
//	                      own warp.VerifyPulsar. Proves the verify integration.
//
// The binary deliberately makes the honest status impossible to miss: production
// threshold signing does not work yet, and the only path that produces a
// signature is the clearly-labeled non-threshold reference.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/luxfi/ids"
	"github.com/luxfi/pulsard"
	"github.com/luxfi/warp"
)

func main() {
	mode := flag.String("mode", "threshold", "threshold (real, fail-closed) | reference (dev/test footgun)")
	subjectHex := flag.String("subject", "", "32-byte subject as hex; default = sha256(\"pulsard-demo\")")
	flag.Parse()

	subject := demoSubject()
	if *subjectHex != "" {
		b, err := hex.DecodeString(*subjectHex)
		if err != nil || len(b) != pulsard.SubjectLen {
			fmt.Fprintf(os.Stderr, "subject must be %d bytes of hex\n", pulsard.SubjectLen)
			os.Exit(2)
		}
		subject = b
	}

	switch *mode {
	case "threshold":
		os.Exit(runThreshold(subject))
	case "reference":
		os.Exit(runReference(subject))
	default:
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", *mode)
		os.Exit(2)
	}
}

// runThreshold exercises the production path: the default fail-closed engine. It
// is expected to refuse, honestly, with ErrThresholdMLDSAUnimplemented.
func runThreshold(subject []byte) int {
	era := mustDemoEra()
	signer, err := pulsard.New(era) // default Unimplemented engine
	if err != nil {
		fmt.Fprintln(os.Stderr, "new signer:", err)
		return 1
	}
	if _, err := signer.ThresholdSign(subject); err != nil {
		fmt.Println("threshold signing is FAIL-CLOSED (as designed):")
		fmt.Println("  ", err)
		fmt.Println("wire a sound ThresholdEngine before production use (see docs/talus-design.tex)")
		return 1
	}
	fmt.Println("unexpected: threshold signing returned a signature with no engine")
	return 1
}

// runReference runs the dev/test footgun and proves the verify path.
func runReference(subject []byte) int {
	fmt.Println("!! reference mode is a DEV/TEST FOOTGUN — single party holds the group secret; NOT threshold")
	signer, era, err := pulsard.NewReferenceSigner(
		demoID("chain"), demoID("signer-set"), 1, 0, 0,
		warp.WeightThreshold{Numerator: 2, Denominator: 3}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reference signer:", err)
		return 1
	}
	ev, err := signer.ThresholdSign(subject)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		return 1
	}
	// Independent confirmation via the chain's verifier.
	if err := warp.VerifyPulsar(ev, subject, era); err != nil {
		fmt.Fprintln(os.Stderr, "VERIFY FAILED:", err)
		return 1
	}
	pkDigest := sha256.Sum256(era.MLDSAPubKey)
	fmt.Printf("subject       : %s\n", hex.EncodeToString(subject))
	fmt.Printf("group pk sha256: %s\n", hex.EncodeToString(pkDigest[:]))
	fmt.Printf("suite         : %s\n", ev.SuiteID)
	fmt.Printf("signature size: %d bytes\n", len(ev.Signature))
	fmt.Println("VERIFIED via warp.VerifyPulsar ✓")
	return 0
}

func demoSubject() []byte {
	d := sha256.Sum256([]byte("pulsard-demo"))
	return d[:]
}

func demoID(label string) ids.ID {
	var x ids.ID
	copy(x[:], label)
	return x
}

func mustDemoEra() warp.PulsarKeyEra {
	// A structurally valid era (real group key) so New() succeeds and the
	// fail-closed behavior comes from the ENGINE, not a bad era.
	_, era, err := pulsard.NewReferenceDealer(
		demoID("chain"), demoID("signer-set"), 1, 0, 0,
		warp.WeightThreshold{Numerator: 2, Denominator: 3}, nil)
	if err != nil {
		panic(err)
	}
	return era
}
