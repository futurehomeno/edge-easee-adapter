package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

const sealedPrefix = "enc:"

// secretKey is in the public source: it keeps the password out of a casual read or a copied
// file, not away from someone holding both the file and the binary.
var secretKey, _ = hex.DecodeString("b298bc32211dbfbd8391cee6eb6a770a1aca925e2ea6facfb6a3c71d4465bd6a")

// Secret is a string stored encrypted in JSON. A value without the prefix is plain text written
// before encryption and is read as is.
type Secret string

func (s Secret) MarshalJSON() ([]byte, error) {
	if s == "" {
		return json.Marshal("")
	}

	gcm, err := secretCipher()
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return json.Marshal(sealedPrefix + base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(s), nil)))
}

func (s *Secret) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	sealed, ok := strings.CutPrefix(value, sealedPrefix)
	if !ok {
		*s = Secret(value)

		return nil
	}

	gcm, err := secretCipher()
	if err != nil {
		return err
	}

	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil || len(raw) < gcm.NonceSize() {
		return errors.New("malformed encrypted secret")
	}

	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return errors.New("decrypt secret failed")
	}

	*s = Secret(plain)

	return nil
}

func secretCipher() (cipher.AEAD, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}
