package data

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdCryptoHashModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.crypto.hash",
		Path: "std.crypto.hash",
		Exports: map[string]runtime.Value{
			"aesGCMDecrypt": &runtime.NativeFunction{Name: "aesGCMDecrypt", Arity: 3, Fn: cryptoAESGCMDecrypt},
			"aesGCMEncrypt": &runtime.NativeFunction{Name: "aesGCMEncrypt", Arity: 2, Fn: cryptoAESGCMEncrypt},
			"base64Decode":  &runtime.NativeFunction{Name: "base64Decode", Arity: 1, Fn: cryptoBase64Decode},
			"base64Encode":  &runtime.NativeFunction{Name: "base64Encode", Arity: 1, Fn: cryptoBase64Encode},
			"hexDecode":     &runtime.NativeFunction{Name: "hexDecode", Arity: 1, Fn: cryptoHexDecode},
			"hexEncode":     &runtime.NativeFunction{Name: "hexEncode", Arity: 1, Fn: cryptoHexEncode},
			"hmacSHA256":    &runtime.NativeFunction{Name: "hmacSHA256", Arity: 2, Fn: cryptoHMACSHA256},
			"hmacSHA512":    &runtime.NativeFunction{Name: "hmacSHA512", Arity: 2, Fn: cryptoHMACSHA512},
			"randomBytes":   &runtime.NativeFunction{Name: "randomBytes", Arity: 1, Fn: cryptoRandomBytes},
			"sha256":        &runtime.NativeFunction{Name: "sha256", Arity: 1, Fn: cryptoSHA256},
			"sha512":        &runtime.NativeFunction{Name: "sha512", Arity: 1, Fn: cryptoSHA512},
		},
		Done: true,
	}
}

func cryptoSHA256(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("sha256", args[0])
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(text))
	return runtime.StringValue{Value: hex.EncodeToString(sum[:])}, nil
}

func cryptoSHA512(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("sha512", args[0])
	if err != nil {
		return nil, err
	}
	sum := sha512.Sum512([]byte(text))
	return runtime.StringValue{Value: hex.EncodeToString(sum[:])}, nil
}

func cryptoHMACSHA256(args []runtime.Value) (runtime.Value, error) {
	key, text, err := requireStringPair("hmacSHA256", args[0], args[1])
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(text))
	return runtime.StringValue{Value: hex.EncodeToString(mac.Sum(nil))}, nil
}

func cryptoHMACSHA512(args []runtime.Value) (runtime.Value, error) {
	key, text, err := requireStringPair("hmacSHA512", args[0], args[1])
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha512.New, []byte(key))
	mac.Write([]byte(text))
	return runtime.StringValue{Value: hex.EncodeToString(mac.Sum(nil))}, nil
}

func cryptoBase64Encode(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("base64Encode", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: base64.StdEncoding.EncodeToString([]byte(text))}, nil
}

func cryptoBase64Decode(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("base64Decode", args[0])
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func cryptoHexEncode(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("hexEncode", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: hex.EncodeToString([]byte(text))}, nil
}

func cryptoHexDecode(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("hexDecode", args[0])
	if err != nil {
		return nil, err
	}
	data, err := hex.DecodeString(text)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func cryptoRandomBytes(args []runtime.Value) (runtime.Value, error) {
	length, err := requireNonNegativeIntArg("randomBytes", args[0])
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: base64.StdEncoding.EncodeToString(data)}, nil
}

func cryptoAESGCMEncrypt(args []runtime.Value) (runtime.Value, error) {
	key, plaintext, err := requireStringPair("aesGCMEncrypt", args[0], args[1])
	if err != nil {
		return nil, err
	}
	gcm, err := newAESGCM("aesGCMEncrypt", key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"nonce":      runtime.StringValue{Value: base64.StdEncoding.EncodeToString(nonce)},
		"ciphertext": runtime.StringValue{Value: base64.StdEncoding.EncodeToString(ciphertext)},
	}}, nil
}

func cryptoAESGCMDecrypt(args []runtime.Value) (runtime.Value, error) {
	key, err := utils.RequireStringArg("aesGCMDecrypt", args[0])
	if err != nil {
		return nil, err
	}
	nonceText, err := utils.RequireStringArg("aesGCMDecrypt", args[1])
	if err != nil {
		return nil, err
	}
	ciphertextText, err := utils.RequireStringArg("aesGCMDecrypt", args[2])
	if err != nil {
		return nil, err
	}
	gcm, err := newAESGCM("aesGCMDecrypt", key)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceText)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextText)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(plaintext)}, nil
}

func newAESGCM(name, key string) (cipher.AEAD, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("%s expects key length 16, 24, or 32 bytes", name)
	}
	return cipher.NewGCM(block)
}

func requireStringPair(name string, first runtime.Value, second runtime.Value) (string, string, error) {
	left, err := utils.RequireStringArg(name, first)
	if err != nil {
		return "", "", err
	}
	right, err := utils.RequireStringArg(name, second)
	if err != nil {
		return "", "", err
	}
	return left, right, nil
}

func requireNonNegativeIntArg(name string, v runtime.Value) (int, error) {
	value, ok := v.(runtime.IntValue)
	if !ok {
		return 0, fmt.Errorf("%s expects int argument", name)
	}
	if value.Value < 0 {
		return 0, fmt.Errorf("%s expects non-negative int argument", name)
	}
	return int(value.Value), nil
}
