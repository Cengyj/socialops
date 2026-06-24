package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/Wei-Shaw/socialops/internal/service"
	"golang.org/x/crypto/pbkdf2"
)

const (
	twitterExecutionKeyPart1 = "TwR00t_K3y_"
	twitterExecutionKeyPart2 = "P@rt2_S3cr3t"

	twitterExecutionBlockSize = 256
	twitterExecutionLCGA      = 1103515245
	twitterExecutionLCGC      = 12345

	twitterExecutionOuterPassSegment1 = "Tw!tR00t#2025"
	twitterExecutionOuterPassSegment2 = "@Enc$K3y_Pr0t"
	twitterExecutionOuterPassSegment3 = "3ct!0n_L@y3r"
	twitterExecutionPBKDF2Iterations  = 50000
)

var (
	twitterExecutionChaChaConstants = [4]uint32{
		0x61707865,
		0x3320646e,
		0x79622d32,
		0x6b206574,
	}
	twitterExecutionOuterSaltPart1 = []byte{0x7A, 0x3F, 0xE2, 0x91, 0xC8, 0x5D, 0x4B, 0xA6}
	twitterExecutionOuterSaltPart2 = []byte{0x1F, 0x8E, 0x73, 0xD4, 0x2C, 0x9A, 0x56, 0xB0}
)

const twitterExecutionBase85Alphabet = "0123456789" +
	"ABCDEFGHIJ" +
	"KLMNOPQRST" +
	"UVWXYZabcd" +
	"efghijklmn" +
	"opqrstuvwx" +
	"yz!#$%&()*" +
	"+-;<=>?@^_" +
	"`{|}~"

// TwitterExecutionAuthEncryptor matches the FlyingBird Twitter token
// encryption format while encrypting only SocialOps execution_auth payloads.
type TwitterExecutionAuthEncryptor struct {
}

func NewTwitterExecutionAuthEncryptor() service.ExecutionAuthEncryptor {
	return &TwitterExecutionAuthEncryptor{}
}

func (e *TwitterExecutionAuthEncryptor) Encrypt(plaintext string) (string, error) {
	return encryptTwitterExecutionAuthLegacy(plaintext)
}

func (e *TwitterExecutionAuthEncryptor) Decrypt(ciphertext string) (string, error) {
	return decryptTwitterExecutionAuthLegacy(ciphertext)
}

func encryptTwitterExecutionAuthLegacy(jsonData string) (string, error) {
	if jsonData == "" {
		return "", fmt.Errorf("twitter execution auth data cannot be empty")
	}
	key := deriveTwitterExecutionKey(generateTwitterExecutionMasterKey())
	nonce := deriveTwitterExecutionNonce(key)
	encrypted, err := twitterExecutionChaCha20XOR([]byte(jsonData), key, nonce)
	if err != nil {
		return "", err
	}
	shuffled := shuffleTwitterExecutionBlocks(encrypted, key)
	encoded := encodeTwitterExecutionBase85(shuffled)
	return encryptTwitterExecutionOuterAES(encoded)
}

func decryptTwitterExecutionAuthLegacy(encrypted string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("twitter execution auth data cannot be empty")
	}
	decoded, err := decryptTwitterExecutionOuterAES(encrypted)
	if err != nil {
		return "", err
	}
	unencoded, err := decodeTwitterExecutionBase85(decoded)
	if err != nil {
		return "", err
	}
	key := deriveTwitterExecutionKey(generateTwitterExecutionMasterKey())
	unshuffled, err := unshuffleTwitterExecutionBlocks(unencoded, key)
	if err != nil {
		return "", err
	}
	nonce := deriveTwitterExecutionNonce(key)
	plainBytes, err := twitterExecutionChaCha20XOR(unshuffled, key, nonce)
	if err != nil {
		return "", err
	}
	return string(plainBytes), nil
}

func generateTwitterExecutionMasterKey() string {
	part3 := generateTwitterExecutionKeyPart3(twitterExecutionKeyPart1, twitterExecutionKeyPart2)
	return twitterExecutionKeyPart1 + twitterExecutionKeyPart2 + part3
}

func generateTwitterExecutionKeyPart3(part1, part2 string) string {
	round1 := sha256.Sum256([]byte(part1 + part2))
	combined := append(round1[:], []byte(part2)...)
	round2 := sha256.Sum256(combined)
	combined = append(round2[:], []byte(part1)...)
	round3 := sha256.Sum256(combined)
	folded := make([]byte, 16)
	for i := 0; i < 16; i++ {
		folded[i] = round3[i] ^ round3[i+16]
	}
	round4 := sha256.Sum256(folded)
	part3 := base64.URLEncoding.EncodeToString(round4[:12])
	part3 = strings.TrimRight(part3, "=")
	return "_" + part3
}

func deriveTwitterExecutionKey(masterKey string) []byte {
	hash := sha256.Sum256([]byte(masterKey))
	return hash[:]
}

func deriveTwitterExecutionNonce(key []byte) []byte {
	nonce := make([]byte, 12)
	for i := 0; i < 12; i++ {
		nonce[i] = key[i+20] ^ key[i]
	}
	return nonce
}

func twitterExecutionChaCha20XOR(data []byte, key []byte, nonce []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("twitter execution auth key must be 32 bytes, got %d", len(key))
	}
	if len(nonce) != 12 {
		return nil, fmt.Errorf("twitter execution auth nonce must be 12 bytes, got %d", len(nonce))
	}

	result := make([]byte, len(data))
	var state [16]uint32
	state[0] = twitterExecutionChaChaConstants[0]
	state[1] = twitterExecutionChaChaConstants[1]
	state[2] = twitterExecutionChaChaConstants[2]
	state[3] = twitterExecutionChaChaConstants[3]
	for i := 0; i < 8; i++ {
		state[4+i] = binary.LittleEndian.Uint32(key[i*4 : (i+1)*4])
	}
	state[12] = 0
	for i := 0; i < 3; i++ {
		state[13+i] = binary.LittleEndian.Uint32(nonce[i*4 : (i+1)*4])
	}

	offset := 0
	for offset < len(data) {
		keystream := twitterExecutionChaCha20Block(state)
		blockSize := 64
		if len(data)-offset < blockSize {
			blockSize = len(data) - offset
		}
		for i := 0; i < blockSize; i++ {
			keystreamByteIndex := i / 4
			keystreamByteOffset := i % 4
			keystreamByte := byte(keystream[keystreamByteIndex] >> (keystreamByteOffset * 8))
			result[offset+i] = data[offset+i] ^ keystreamByte
		}
		offset += blockSize
		state[12]++
	}
	return result, nil
}

func twitterExecutionChaCha20Block(input [16]uint32) [16]uint32 {
	output := input
	for i := 0; i < 24; i += 2 {
		twitterExecutionQuarterRound(&output[0], &output[4], &output[8], &output[12])
		twitterExecutionQuarterRound(&output[1], &output[5], &output[9], &output[13])
		twitterExecutionQuarterRound(&output[2], &output[6], &output[10], &output[14])
		twitterExecutionQuarterRound(&output[3], &output[7], &output[11], &output[15])
		twitterExecutionQuarterRound(&output[0], &output[5], &output[10], &output[15])
		twitterExecutionQuarterRound(&output[1], &output[6], &output[11], &output[12])
		twitterExecutionQuarterRound(&output[2], &output[7], &output[8], &output[13])
		twitterExecutionQuarterRound(&output[3], &output[4], &output[9], &output[14])
	}
	for i := 0; i < 16; i++ {
		output[i] += input[i]
	}
	return output
}

func twitterExecutionQuarterRound(a, b, c, d *uint32) {
	*a += *b
	*d ^= *a
	*d = twitterExecutionRotateLeft(*d, 16)
	*c += *d
	*b ^= *c
	*b = twitterExecutionRotateLeft(*b, 12)
	*a += *b
	*d ^= *a
	*d = twitterExecutionRotateLeft(*d, 8)
	*c += *d
	*b ^= *c
	*b = twitterExecutionRotateLeft(*b, 7)
}

func twitterExecutionRotateLeft(x uint32, n int) uint32 {
	return (x << n) | (x >> (32 - n))
}

func shuffleTwitterExecutionBlocks(data []byte, key []byte) []byte {
	if len(data) < twitterExecutionBlockSize {
		result := make([]byte, len(data))
		copy(result, data)
		xorTwitterExecutionWithKey(result, key)
		return prependTwitterExecutionCRC32(data, result)
	}

	totalBlocks := (len(data) + twitterExecutionBlockSize - 1) / twitterExecutionBlockSize
	blocks := make([][]byte, totalBlocks)
	for i := 0; i < totalBlocks; i++ {
		start := i * twitterExecutionBlockSize
		end := start + twitterExecutionBlockSize
		if end > len(data) {
			end = len(data)
		}
		blocks[i] = append([]byte(nil), data[start:end]...)
	}

	seed := twitterExecutionShuffleSeed(key)
	for i := totalBlocks - 1; i > 0; i-- {
		seed = seed*twitterExecutionLCGA + twitterExecutionLCGC
		j := int(seed % uint32(i+1))
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	shuffled := make([]byte, 0, len(data))
	for _, block := range blocks {
		shuffled = append(shuffled, block...)
	}
	xorTwitterExecutionWithKey(shuffled, key)
	return prependTwitterExecutionCRC32(data, shuffled)
}

func unshuffleTwitterExecutionBlocks(data []byte, key []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("twitter execution auth data is too short for crc32")
	}
	storedCRC := binary.LittleEndian.Uint32(data[0:4])
	shuffled := append([]byte(nil), data[4:]...)
	if len(shuffled) < twitterExecutionBlockSize {
		xorTwitterExecutionWithKey(shuffled, key)
		if crc32.ChecksumIEEE(shuffled) != storedCRC {
			return nil, fmt.Errorf("twitter execution auth crc32 check failed")
		}
		return shuffled, nil
	}

	xorTwitterExecutionWithKey(shuffled, key)
	dataLen := len(shuffled)
	totalBlocks := (dataLen + twitterExecutionBlockSize - 1) / twitterExecutionBlockSize
	lastBlockSize := dataLen % twitterExecutionBlockSize
	if lastBlockSize == 0 {
		lastBlockSize = twitterExecutionBlockSize
	}
	originalBlockSizes := make([]int, totalBlocks)
	for i := 0; i < totalBlocks-1; i++ {
		originalBlockSizes[i] = twitterExecutionBlockSize
	}
	originalBlockSizes[totalBlocks-1] = lastBlockSize

	perm := make([]int, totalBlocks)
	for i := range perm {
		perm[i] = i
	}
	seed := twitterExecutionShuffleSeed(key)
	for i := totalBlocks - 1; i > 0; i-- {
		seed = seed*twitterExecutionLCGA + twitterExecutionLCGC
		j := int(seed % uint32(i+1))
		perm[i], perm[j] = perm[j], perm[i]
	}

	shuffledBlockSizes := make([]int, totalBlocks)
	for i := 0; i < totalBlocks; i++ {
		shuffledBlockSizes[i] = originalBlockSizes[perm[i]]
	}
	blocks := make([][]byte, totalBlocks)
	offset := 0
	for i := 0; i < totalBlocks; i++ {
		size := shuffledBlockSizes[i]
		if offset+size > len(shuffled) {
			return nil, fmt.Errorf("twitter execution auth shuffled block size is invalid")
		}
		blocks[i] = append([]byte(nil), shuffled[offset:offset+size]...)
		offset += size
	}

	invPerm := make([]int, totalBlocks)
	for i := 0; i < totalBlocks; i++ {
		invPerm[perm[i]] = i
	}
	result := make([]byte, 0, dataLen)
	for i := 0; i < totalBlocks; i++ {
		result = append(result, blocks[invPerm[i]]...)
	}
	if crc32.ChecksumIEEE(result) != storedCRC {
		return nil, fmt.Errorf("twitter execution auth crc32 check failed")
	}
	return result, nil
}

func twitterExecutionShuffleSeed(key []byte) uint32 {
	var seed uint32
	for i := 0; i < 32; i++ {
		seed = (seed << 8) | uint32(key[i])
	}
	return seed
}

func xorTwitterExecutionWithKey(data []byte, key []byte) {
	for i := 0; i < len(data); i++ {
		data[i] ^= key[i%32]
	}
}

func prependTwitterExecutionCRC32(original []byte, transformed []byte) []byte {
	crcBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(crcBytes, crc32.ChecksumIEEE(original))
	output := make([]byte, 4+len(transformed))
	copy(output[0:4], crcBytes)
	copy(output[4:], transformed)
	return output
}

func encodeTwitterExecutionBase85(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var result strings.Builder
	i := 0
	for i < len(data) {
		var value uint64
		count := 0
		for j := 0; j < 4 && i < len(data); j++ {
			value = (value << 8) | uint64(data[i])
			i++
			count++
		}
		if count < 4 {
			value <<= uint((4 - count) * 8)
		}
		chars := make([]byte, 5)
		tempVal := value
		for j := 4; j >= 0; j-- {
			chars[j] = twitterExecutionBase85Alphabet[tempVal%85]
			tempVal /= 85
		}
		if count == 4 {
			result.Write(chars)
		} else {
			result.Write(chars[:count+1])
		}
	}
	return result.String()
}

func decodeTwitterExecutionBase85(encoded string) ([]byte, error) {
	if len(encoded) == 0 {
		return nil, nil
	}
	reverseMap := make(map[byte]uint64, len(twitterExecutionBase85Alphabet))
	for i, c := range []byte(twitterExecutionBase85Alphabet) {
		reverseMap[c] = uint64(i)
	}

	var result []byte
	i := 0
	for i < len(encoded) {
		var value uint64
		count := 0
		for j := 0; j < 5 && i < len(encoded); j++ {
			idx, ok := reverseMap[encoded[i]]
			if !ok {
				return nil, fmt.Errorf("invalid twitter execution auth base85 character: %c", encoded[i])
			}
			value = value*85 + idx
			i++
			count++
		}
		if count == 5 {
			result = append(result, byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
			continue
		}
		outputBytes := count - 1
		for j := 0; j < 5-count; j++ {
			value = value*85 + 84
		}
		shift := uint((4 - outputBytes) * 8)
		value >>= shift
		bytes := make([]byte, outputBytes)
		for j := outputBytes - 1; j >= 0; j-- {
			bytes[j] = byte(value & 0xFF)
			value >>= 8
		}
		result = append(result, bytes...)
	}
	return result, nil
}

func encryptTwitterExecutionOuterAES(data string) (string, error) {
	plainBytes := []byte(data)
	key, iv := twitterExecutionOuterKeyAndIV()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create twitter execution auth outer aes cipher: %w", err)
	}
	paddedData := pkcs7PadTwitterExecution(plainBytes, aes.BlockSize)
	ciphertext := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedData)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptTwitterExecutionOuterAES(base64Data string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("decode twitter execution auth outer aes base64: %w", err)
	}
	if len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return "", fmt.Errorf("twitter execution auth outer aes ciphertext length is invalid")
	}
	key, iv := twitterExecutionOuterKeyAndIV()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create twitter execution auth outer aes cipher: %w", err)
	}
	plaintext := make([]byte, len(encrypted))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, encrypted)
	unpadded, err := pkcs7UnpadTwitterExecution(plaintext, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(unpadded), nil
}

func twitterExecutionOuterKeyAndIV() ([]byte, []byte) {
	salt := append([]byte(nil), twitterExecutionOuterSaltPart1...)
	salt = append(salt, twitterExecutionOuterSaltPart2...)
	password := twitterExecutionOuterPassSegment1 + twitterExecutionOuterPassSegment2 + twitterExecutionOuterPassSegment3
	keyAndIV := pbkdf2.Key([]byte(password), salt, twitterExecutionPBKDF2Iterations, 48, sha256.New)
	return keyAndIV[:32], keyAndIV[32:]
}

func pkcs7PadTwitterExecution(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

func pkcs7UnpadTwitterExecution(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("twitter execution auth outer aes plaintext is empty")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, fmt.Errorf("twitter execution auth outer aes padding is invalid")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, fmt.Errorf("twitter execution auth outer aes padding is invalid")
		}
	}
	return data[:len(data)-padding], nil
}
