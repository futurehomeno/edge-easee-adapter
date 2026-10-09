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

	log "github.com/sirupsen/logrus"
)

const sealedPrefix = "enc:"

// The key is in the public source: it keeps the password out of a casual read or a copied
// file, not away from someone holding both the file and the binary.
var secretAEAD = func() cipher.AEAD {
	key, err := hex.DecodeString("b298bc32211dbfbd8391cee6eb6a770a1aca925e2ea6facfb6a3c71d4465bd6a")
	if err != nil {
		panic(err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	return gcm
}()

// Secret is a string stored encrypted in JSON. A value without the prefix is plain text written
// before encryption and is read as is.
type Secret string

func (s Secret) MarshalJSON() ([]byte, error) {
	if s == "" {
		return json.Marshal("")
	}

	nonce := make([]byte, secretAEAD.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return json.Marshal(sealedPrefix + base64.StdEncoding.EncodeToString(secretAEAD.Seal(nonce, nonce, []byte(s), nil)))
}

// UnmarshalJSON reads a value it cannot decrypt as empty rather than failing: the whole secrets
// file would fail with it, and the tokens next to it still hold a session.
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

	plain, err := open(sealed)
	if err != nil {
		log.Warnf("[config] Drop a stored secret that does not decrypt. err: %v", err)
	}

	*s = Secret(plain)

	return nil
}

func open(sealed string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return nil, err
	}

	if len(raw) < secretAEAD.NonceSize() {
		return nil, errors.New("shorter than the nonce")
	}

	return secretAEAD.Open(nil, raw[:secretAEAD.NonceSize()], raw[secretAEAD.NonceSize():], nil)
}
