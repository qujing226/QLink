package client

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qujing226/QLink/spec/gen"
	"github.com/qujing226/QLink/spec/pkg/blockchain"
	"github.com/qujing226/QLink/spec/pkg/connect"
	"github.com/qujing226/QLink/spec/pkg/secure"
)

// Session 维护与特定 Peer 的加密会话状态
type Session struct {
	PeerDid   string
	TxRatchet *secure.ChainKey // 我发给对方
	RxRatchet *secure.ChainKey // 对方发给我
	TxSeq     uint64
}

type Client struct {
	Did      string
	SignKeys *secure.SignKeyPair
	KemKeys  *secure.KyberKeyPair

	Chain     *blockchain.OptimisticCache
	RelayAddr string
	Conn      net.Conn
	
	// 状态管理
	mu             sync.Mutex
	CurrentSession *Session
	handshakeChan  chan error // 用于通知握手结果
	
	// 用户回调
	OnMessage func(sender string, msg []byte)
}

func NewClient(did string, chain *blockchain.OptimisticCache, relayAddr string) (*Client, error) {
	signKp, err := secure.NewSignKeyPair()
	if err != nil {
		return nil, err
	}
	kemKp, err := secure.NewKyberKeyPair()
	if err != nil {
		return nil, err
	}

	// 注册 DID (Simulated)
	pk, _ := kemKp.Export()
	signPk, _ := signKp.Export()
	doc := make([]byte, len(signPk)+len(pk))
	copy(doc[0:], signPk)
	copy(doc[len(signPk):], pk)
	chain.RegisterDidDoc(did, doc)

	return &Client{
		Did:           did,
		SignKeys:      signKp,
		KemKeys:       kemKp,
		Chain:         chain,
		RelayAddr:     relayAddr,
		handshakeChan: make(chan error, 1),
	}, nil
}

func (c *Client) Start() error {
	conn, err := net.Dial("tcp", c.RelayAddr)
	if err != nil {
		return err
	}
	c.Conn = conn
	fmt.Printf("[%s] Connected to Relay at %s\n", c.Did, c.RelayAddr)

	// Register
	regPkt := &didproto.Packet{
		Header: c.newHeader(""),
		Payload: &didproto.Packet_Status{
			Status: &didproto.Status{Code: didproto.Status_SUCCESS, Message: "Register"},
		},
	}
	if err := connect.WritePacket(c.Conn, regPkt); err != nil {
		return err
	}

	go c.readLoop()
	return nil
}

func (c *Client) readLoop() {
	defer c.Conn.Close()
	for {
		pkt, err := connect.ReadPacket(c.Conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[%s] Read error: %v\n", c.Did, err)
			}
			return
		}
		c.handlePacket(pkt)
	}
}

func (c *Client) handlePacket(pkt *didproto.Packet) {
	switch p := pkt.Payload.(type) {
	case *didproto.Packet_KemInit:
		c.handleKemInit(pkt.Header, p.KemInit)
	case *didproto.Packet_KemConfirm:
		c.handleKemConfirm(pkt.Header, p.KemConfirm)
	case *didproto.Packet_SecureMessage:
		c.handleSecureMessage(pkt.Header, p.SecureMessage)
	case *didproto.Packet_Status:
		// 简单的状态打印
		if p.Status.Code != didproto.Status_SUCCESS {
			fmt.Printf("[%s] ERROR from %s: %s\n", c.Did, pkt.Header.FromDid, p.Status.Message)
		}
	}
}

// =============================================================================
//  Core Logic: Handshake (Initiator)
// =============================================================================

func (c *Client) Handshake(targetDid string) error {
	// 1. Optimistic Cache Resolve
	doc, err := c.Chain.Resolve(targetDid)
	if err != nil {
		return fmt.Errorf("resolve failed: %w", err)
	}
	if len(doc) < 32 {
		return errors.New("invalid doc")
	}
	targetKyberPk, err := secure.LoadFromBytes(doc[32:], nil)
	if err != nil {
		return err
	}

	// 2. Encapsulate (Kyber)
	ct, ss, err := targetKyberPk.Encapsulate()
	if err != nil {
		return err
	}

	// 3. Prepare Session (Pending)
	// Derive Root Key -> Chain Keys
	// Rule: Initiator Tx = HKDF(SS, "A->B")
	//       Initiator Rx = HKDF(SS, "B->A")
	txRoot := secure.SimpleKDF(ss, nil, []byte("A->B"))
	rxRoot := secure.SimpleKDF(ss, nil, []byte("B->A"))
	
	c.mu.Lock()
	c.CurrentSession = &Session{
		PeerDid:   targetDid,
		TxRatchet: secure.NewChainKey(txRoot),
		RxRatchet: secure.NewChainKey(rxRoot),
	}
	c.mu.Unlock()

	// 4. Send KEMInit
	nonce := make([]byte, 32)
	rand.Read(nonce)
	
	// TODO: Sign the payload (omitted for brevity, use secure.Sign)
	sig, _ := c.SignKeys.Sign(append(ct, nonce...))

	pkt := &didproto.Packet{
		Header: c.newHeader(targetDid),
		Payload: &didproto.Packet_KemInit{
			KemInit: &didproto.KEMInit{
				Ct:        ct,
				Nonce:     nonce,
				Signature: sig,
			},
		},
	}
	
	if err := connect.WritePacket(c.Conn, pkt); err != nil {
		return err
	}

	// 5. Wait for KEMConfirm
	select {
	case err := <-c.handshakeChan:
		return err
	case <-time.After(5 * time.Second):
		return errors.New("handshake timeout")
	}
}

// =============================================================================
//  Core Logic: Handshake (Responder)
// =============================================================================

func (c *Client) handleKemInit(header *didproto.Header, body *didproto.KEMInit) {
	// 1. Decapsulate
	ss, err := c.KemKeys.Decapsulate(body.Ct)
	if err != nil {
		fmt.Printf("[%s] Decap failed: %v\n", c.Did, err)
		return
	}

	// 2. Setup Session (Mirrored)
	// Responder Tx = HKDF(SS, "B->A")  <-- Matches Initiator Rx
	// Responder Rx = HKDF(SS, "A->B")  <-- Matches Initiator Tx
	txRoot := secure.SimpleKDF(ss, nil, []byte("B->A"))
	rxRoot := secure.SimpleKDF(ss, nil, []byte("A->B"))

	c.mu.Lock()
	c.CurrentSession = &Session{
		PeerDid:   header.FromDid,
		TxRatchet: secure.NewChainKey(txRoot),
		RxRatchet: secure.NewChainKey(rxRoot),
	}
	c.mu.Unlock()

	// 3. Send Confirm
	// TODO: Verify signature of Initiator first!
	
	// Sign nonce hash
	sig, _ := c.SignKeys.Sign(body.Nonce) // Simplified: just sign nonce

	resp := &didproto.Packet{
		Header: c.newHeader(header.FromDid),
		Payload: &didproto.Packet_KemConfirm{
			KemConfirm: &didproto.KEMConfirm{
				NonceHash: body.Nonce, // Echo nonce back
				Signature: sig,
			},
		},
	}
	connect.WritePacket(c.Conn, resp)
	fmt.Printf("[%s] Handshake accepted with %s\n", c.Did, header.FromDid)
}

func (c *Client) handleKemConfirm(header *didproto.Header, body *didproto.KEMConfirm) {
	// 收到对方确认
	// TODO: Check signature
	
	// Unblock Handshake
	select {
	case c.handshakeChan <- nil:
	default:
	}
}

// =============================================================================
//  Core Logic: Secure Messaging (Ratchet)
// =============================================================================

func (c *Client) SendMessage(msg string) error {
	c.mu.Lock()
	sess := c.CurrentSession
	c.mu.Unlock()
	if sess == nil {
		return errors.New("no active session")
	}

	// 1. Ratchet Forward -> Get Message Key
	msgKey, err := sess.TxRatchet.Ratchet()
	if err != nil {
		return err
	}

	// 2. Encrypt (AES-GCM)
	block, _ := aes.NewCipher(msgKey)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	ciphertext := gcm.Seal(nil, nonce, []byte(msg), nil)

	// 3. Send
	sess.TxSeq++
	pkt := &didproto.Packet{
		Header: c.newHeader(sess.PeerDid),
		Payload: &didproto.Packet_SecureMessage{
			SecureMessage: &didproto.SecureMessage{
				SequenceNumber: sess.TxSeq,
				Ciphertext:     ciphertext,
				Nonce:          nonce,
				Tag:            nil, // In Go GCM, tag is appended to ciphertext
			},
		},
	}
	return connect.WritePacket(c.Conn, pkt)
}

func (c *Client) handleSecureMessage(header *didproto.Header, body *didproto.SecureMessage) {
	c.mu.Lock()
	sess := c.CurrentSession
	c.mu.Unlock()
	
	if sess == nil || sess.PeerDid != header.FromDid {
		fmt.Printf("[%s] Drop msg from unknown/wrong peer %s\n", c.Did, header.FromDid)
		return
	}

	// 1. Ratchet Forward -> Get Message Key
	// Note: In real Double Ratchet, we might need to "skip" keys if packets arrive out of order.
	// Here we assume TCP ordered delivery for simplicity.
	msgKey, err := sess.RxRatchet.Ratchet()
	if err != nil {
		return
	}

	// 2. Decrypt
	block, _ := aes.NewCipher(msgKey)
	gcm, _ := cipher.NewGCM(block)
	plaintext, err := gcm.Open(nil, body.Nonce, body.Ciphertext, nil)
	if err != nil {
		fmt.Printf("[%s] Decrypt failed: %v\n", c.Did, err)
		return
	}

	if c.OnMessage != nil {
		c.OnMessage(header.FromDid, plaintext)
	} else {
		fmt.Printf("[%s] MSG from %s: %s\n", c.Did, header.FromDid, string(plaintext))
	}
}

func (c *Client) newHeader(to string) *didproto.Header {
	return &didproto.Header{
		RequestId: uuid.New().String(),
		FromDid:   c.Did,
		ToDid:     to,
		Timestamp: time.Now().UnixMilli(),
	}
}