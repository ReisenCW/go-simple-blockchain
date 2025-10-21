package blockchain

import (
	"bytes"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const walletFile = "wallet_%s.dat"

// Wallets stores a collection of wallets
type Wallets struct {
	Wallets map[string]*Wallet
}

// NewWallets creates Wallets and fills it from a file if it exists
func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile(nodeID)

	return &wallets, err
}

// CreateWallet adds a Wallet to Wallets
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = wallet

	return address
}

// GetAddresses returns an array of addresses stored in the wallet file
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// GetWallet returns a Wallet by its address
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// LoadFromFile loads wallets from the file
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	// decode stored map of address -> {privateKeyDER, publicKey}
	var stored map[string]struct {
		PrivateKey []byte
		PublicKey  []byte
	}
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	if err := decoder.Decode(&stored); err != nil {
		log.Panic(err)
	}

	for addr, sw := range stored {
		privKey, err := x509.ParseECPrivateKey(sw.PrivateKey)
		if err != nil {
			log.Panic(err)
		}
		ws.Wallets[addr] = &Wallet{*privKey, sw.PublicKey}
	}

	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveToFile(nodeID string) {
	// prepare a simple map[address] -> {privateKeyDER, publicKey}
	stored := make(map[string]struct {
		PrivateKey []byte
		PublicKey  []byte
	})

	for addr, w := range ws.Wallets {
		der, err := x509.MarshalECPrivateKey(&w.PrivateKey)
		if err != nil {
			log.Panic(err)
		}
		stored[addr] = struct {
			PrivateKey []byte
			PublicKey  []byte
		}{der, w.PublicKey}
	}

	var content bytes.Buffer
	encoder := gob.NewEncoder(&content)
	if err := encoder.Encode(stored); err != nil {
		log.Panic(err)
	}

	walletFile := fmt.Sprintf(walletFile, nodeID)
	if err := os.WriteFile(walletFile, content.Bytes(), 0644); err != nil {
		log.Panic(err)
	}
}
