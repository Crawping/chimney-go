package privacy

import (
	"bytes"
	"chimney-go/utils"
	"crypto/rand"
	"errors"
	"io"
	"log"

	"golang.org/x/crypto/chacha20poly1305"
)

type ploy struct {
	iv []byte
}

const (
	ployName = "CHACHA-Ploy1305"
	ployCode = 0x1236
)

func (p *ploy) Compress(src []byte, key []byte) ([]byte, error) {
	defer utils.Trace("Compress")()

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		log.Println("key of chacha20Poly1305 is invalid!")
		return nil, err
	}

	ciphertext := aead.Seal(nil, p.iv, src, nil)
	if len(ciphertext) == 0 {
		return nil, errors.New("compressed failed")
	}

	return ciphertext, nil
}

func (p *ploy) Uncompress(src []byte, key []byte) ([]byte, error) {
	defer utils.Trace("Uncompress")()

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		log.Println("key of chacha20Poly1305 is invalid!(uncompress)")
		return nil, err
	}

	plaintext, err := aead.Open(nil, p.iv, src, nil)
	if len(plaintext) == 0 {
		return nil, errors.New("compressed failed")
	}
	return plaintext, err
}

func (p *ploy) MakeSalt() []byte {
	nonce := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil
	}
	return nonce
}

func (p *ploy) GetIV() []byte {
	return p.iv
}

func (p *ploy) SetIV(iv []byte) {
	p.iv = make([]byte, len(iv))
	copy(p.iv, iv)
}

func (p *ploy) GetSize() int {
	return 2 + 1 + len(p.iv)
}

func (p *ploy) ToBytes() []byte {
	var op bytes.Buffer
	mask := utils.Uint162Bytes(ployCode)
	op.Write(mask)
	lv := (byte)(len(p.iv))
	op.WriteByte(lv)
	if lv > 0 {
		op.Write(p.iv)
	}
	return op.Bytes()
}

//From bytes
func (p *ploy) FromBytes(v []byte) error {
	op := bytes.NewBuffer(v)
	lvl := op.Next(1)
	if len(lvl) < 1 {
		return errors.New("out of length")
	}

	value := int(lvl[0])
	if value > 0 {
		iv := op.Next(value)
		p.SetIV(iv)
	}
	return nil
}

func init() {
	register(ployName, ployCode, &ploy{})
}
