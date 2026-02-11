package main

import (
	"fmt"
	"time"

	"github.com/qujing226/QLink/spec/pkg/blockchain"
	"github.com/qujing226/QLink/spec/pkg/client"
	"github.com/qujing226/QLink/spec/pkg/server"
)

const (
	RelayPort = "19000"
	RelayAddr = "localhost:" + RelayPort
)

func main() {
	fmt.Println("=== QLink Protocol Simulation Start ===")

	// 1. Start Relay Server
	relay := server.NewRelayServer()
	if err := relay.Start(RelayPort); err != nil {
		panic(err)
	}
	defer relay.Stop()
	time.Sleep(100 * time.Millisecond) // Wait for relay to bind

	// 2. Setup Shared Infrastructure (Blockchain)
	// Latency = 500ms to simulate real-world blockchain slowness
	simChain := blockchain.NewSimulatedChain(500 * time.Millisecond)

	// Alice and Bob share the same view of the blockchain (and cache logic)
	// In reality, they would have separate local caches.
	// Let's give them separate caches wrapping the SAME chain.
	aliceCache := blockchain.NewOptimisticCache(simChain, onMismatch("Alice"))
	bobCache := blockchain.NewOptimisticCache(simChain, onMismatch("Bob"))

	// 3. Initialize Clients
	// Bob (Responder)
	bob, err := client.NewClient("did:qlink:bob", bobCache, RelayAddr)
	if err != nil {
		panic(err)
	}
	bob.OnMessage = func(sender string, msg []byte) {
		fmt.Printf("[Bob] Decrypted MSG from %s: '%s'\n", sender, string(msg))
	}
	if err := bob.Start(); err != nil {
		panic(err)
	}

	// Alice (Initiator)
	alice, err := client.NewClient("did:qlink:alice", aliceCache, RelayAddr)
	if err != nil {
		panic(err)
	}
	alice.OnMessage = func(sender string, msg []byte) {
		fmt.Printf("[Alice] Decrypted MSG from %s: '%s'\n", sender, string(msg))
	}
	if err := alice.Start(); err != nil {
		panic(err)
	}

	time.Sleep(500 * time.Millisecond) // Wait for network stabilization

	// =========================================================================
	// Experiment 1: Cold Start Handshake (Chain Lookup)
	// =========================================================================
	fmt.Println("\n--- Experiment 1: Cold Start Handshake (Expect > 500ms latency) ---")
	start := time.Now()

	if err := alice.Handshake("did:qlink:bob"); err != nil {
		panic(err)
	}

	duration := time.Since(start)
	fmt.Printf(">>> Handshake 1 Finished in %v\n", duration)

	// =========================================================================
	// Experiment 2: Ratchet Communication
	// =========================================================================
	fmt.Println("\n--- Experiment 2: Secure Communication & Ratchet ---")

	// Msg 1
	fmt.Println(">>> Alice sending 'Msg 1'...")
	alice.SendMessage("Hello Bob, this is Message 1")
	time.Sleep(100 * time.Millisecond)

	// Msg 2
	fmt.Println(">>> Alice sending 'Msg 2'...")
	alice.SendMessage("And this is Message 2 (Keys should have evolved)")
	time.Sleep(100 * time.Millisecond)

	// =========================================================================
	// Experiment 3: Optimistic Cache (Session Resumption)
	// =========================================================================
	fmt.Println("\n--- Experiment 3: Warm Handshake (Optimistic Cache) ---")
	// Simulate a new handshake request (e.g., previous session expired or new device)
	// Alice already has Bob's doc in her cache from Exp 1.

	start = time.Now()

	if err := alice.Handshake("did:qlink:bob"); err != nil {
		panic(err)
	}

	duration = time.Since(start)
	fmt.Printf(">>> Handshake 2 Finished in %v (Should be near instant)\n", duration)

	fmt.Println("\n=== Simulation Complete ===")
}

func onMismatch(who string) blockchain.VerificationCallback {
	return func(did string, cached, fresh []byte) {
		fmt.Printf("[%s] ALERT: Blockchain Mismatch detected for %s! Possible Attack!\n", who, did)
	}
}
