package sd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

// LoraHash is a hash of a safetensors file.
// The hash is calculated by skipping the first 8 bytes of the file (headers)
// This is to mimic the behavior of [kohya-ss hash for safetensors] and [auto]'s implementation
// The AutoV3 hash is usually the hash that's printed which is only the first 12 bytes of the SHA256 hash.
//
// [kohya-ss hash for safetensors]: https://github.com/kohya-ss/sd-scripts/blob/main/library/train_util.py
// [auto]: https://github.com/AUTOMATIC1111/stable-diffusion-webui/blob/adadb4e3c7382bf3e4f7519126cd6c70f4f8557b/modules/hashes.py#L69-L83
type LoraHash struct {
	AutoV2     *string // Deprecated: first 10 bytes of SHA256. AutoV2 is usually for CheckpointHash.AutoV2
	SHA256     *string // Deprecated: The old implementation where the header is included in the hash
	AutoV3Full string
	AutoV3     string // first 12 bytes of AutoV3Full. AutoV3 is usually the hash that's printed.
}

var ErrNotSafeTensor = errors.New("not a safetensors file")

var ErrEmptyPath = errors.New("empty path")

// LoraSafetensorHash calculates the hash of a lora safetensors file.
// It skips the first 8 bytes of the file (headers) and hashes the rest of the file.
// The AutoV3 hash is usually the hash that's printed which is only the first 12 bytes of the SHA256 hash.
func LoraSafetensorHash(path string) (*LoraHash, error) {
	if path == "" {
		return nil, ErrEmptyPath
	}
	if !strings.HasSuffix(path, ".safetensors") {
		return nil, ErrNotSafeTensor
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	header := make([]byte, 8)
	_, err = file.Read(header)
	if err != nil {
		return nil, err
	}

	n := binary.LittleEndian.Uint64(header)

	offset := int64(n) + 8
	_, err = file.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	buf := make([]byte, 1024*1024)

	for {
		readBytes, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		hasher.Write(buf[:readBytes])
	}

	h := hex.EncodeToString(hasher.Sum(nil))
	return &LoraHash{
		AutoV3Full: h,
		AutoV3:     h[:12],
	}, nil
}

// CheckpointHash is a hash of a safetensors file.
// As opposed to LoraHash, CheckpointHash is calculated by hashing the entire file.
// The AutoV2 hash is usually the hash that's printed which is only the first 10 bytes of the SHA256 hash.
type CheckpointHash struct {
	SHA256 string
	AutoV2 string // first 10 bytes of SHA256. AutoV2 is usually the hash that's printed.
}

// CheckpointSafeTensorHash calculates the hash of a safetensors file.
// It uses io.Copy to copy the file to the hasher.
func CheckpointSafeTensorHash(path string) (*CheckpointHash, error) {
	if path == "" {
		return nil, ErrEmptyPath
	}
	if !strings.HasSuffix(path, ".safetensors") {
		return nil, ErrNotSafeTensor
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return nil, err
	}

	h := hex.EncodeToString(hasher.Sum(nil))
	return &CheckpointHash{
		SHA256: h,
		AutoV2: h[:10],
	}, nil
}

// CalculateHash calculates the hash of a safetensors file.
// If lora is true, it calculates the hash using LoraSafetensorHash, which gives the LoraHash.AutoV3
// Otherwise, it calculates the hash using CheckpointSafeTensorHash, which gives the CheckpointHash.AutoV2
func CalculateHash(path string, lora bool) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}
	if lora {
		hash, err := LoraSafetensorHash(path)
		if err != nil {
			return "", err
		}
		return hash.AutoV3, nil
	}
	hash, err := CheckpointSafeTensorHash(path)
	if err != nil {
		return "", err
	}
	return hash.AutoV2, nil
}
