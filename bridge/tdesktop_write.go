package bridge

import (
	"crypto/aes"
	"crypto/md5" // #nosec G501
	"crypto/rand"
	"crypto/sha1" // #nosec G505
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-faster/errors"
	"github.com/gotd/ige"
	"golang.org/x/crypto/pbkdf2"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/crypto"
	"github.com/gotd/td/session"
)

// tdesktop's own encoder/decoder (github.com/gotd/td/session/tdesktop) is
// read-only and unexported, so the primitives below re-derive the write side
// of the same format. Line references point at the C++ implementation that
// defines it.
const (
	// See https://github.com/telegramdesktop/tdesktop/blob/v2.9.8/Telegram/SourceFiles/storage/details/storage_file_utilities.cpp#L300.
	tdesktopStrongIterations = 100000
	tdesktopSaltSize         = 32

	// See https://github.com/telegramdesktop/tdesktop/blob/dev/Telegram/SourceFiles/main/main_account.cpp#L359.
	tdesktopDbiMtpAuthorization = 0x4b
	tdesktopWideIDsTag          = ^uint64(0)
)

var tdesktopFileMagic = [4]byte{'T', 'D', 'F', '$'}

// TDesktopFiles maps a tdata-root-relative file name to its raw contents.
type TDesktopFiles map[string][]byte

// ToTDesktopFiles converts data into the raw files of a Telegram Desktop
// tdata folder holding a single account. userID is required: tdata embeds it
// alongside the DC/auth key, but Telethon/Pyrogram sessions don't carry it.
// passcode is the local passcode to encrypt the folder with; pass nil for
// the (default) no-passcode case.
//
// The result is returned as an in-memory file map rather than written
// straight to disk, so callers can zip it, ship it over the network, or
// write it wherever they like. ToTDesktop is a convenience wrapper that
// writes it to a directory.
func ToTDesktopFiles(data *session.Data, userID int64, passcode []byte) (TDesktopFiles, error) {
	if err := validateExport(data); err != nil {
		return nil, err
	}

	salt := make([]byte, tdesktopSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, errors.Wrap(err, "generate salt")
	}
	var localKey crypto.Key
	if _, err := rand.Read(localKey[:]); err != nil {
		return nil, errors.Wrap(err, "generate local key")
	}
	passcodeKey := tdesktopLocalKey(passcode, salt)

	keyEncrypted, err := tdesktopEncryptLocal(tdesktopPrefixed(localKey[:]), passcodeKey)
	if err != nil {
		return nil, errors.Wrap(err, "encrypt local key")
	}

	// A single account at index 0.
	var info []byte
	info = binary.BigEndian.AppendUint32(info, 1)
	info = binary.BigEndian.AppendUint32(info, 0)
	infoEncrypted, err := tdesktopEncryptLocal(tdesktopPrefixed(info), localKey)
	if err != nil {
		return nil, errors.Wrap(err, "encrypt account info")
	}

	var keyDataContent []byte
	keyDataContent = tdesktopAppendArray(keyDataContent, salt)
	keyDataContent = tdesktopAppendArray(keyDataContent, keyEncrypted)
	keyDataContent = tdesktopAppendArray(keyDataContent, infoEncrypted)

	auth := tdesktopSerializeAuthorization(uint64(userID), data.DC, data.AuthKey)
	authEncrypted, err := tdesktopEncryptLocal(tdesktopPrefixed(auth), localKey)
	if err != nil {
		return nil, errors.Wrap(err, "encrypt mtp authorization")
	}
	var accountDataContent []byte
	accountDataContent = tdesktopAppendArray(accountDataContent, authEncrypted)

	return TDesktopFiles{
		"key_datas":                   tdesktopWriteFile(keyDataContent),
		tdesktopFileKey("data") + "s": tdesktopWriteFile(accountDataContent),
	}, nil
}

// ToTDesktop writes a Telegram Desktop tdata folder for data into root,
// creating it if necessary. See ToTDesktopFiles for the meaning of userID
// and passcode.
func ToTDesktop(data *session.Data, userID int64, root string, passcode []byte) error {
	files, err := ToTDesktopFiles(data, userID, passcode)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return errors.Wrap(err, "create root")
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), content, 0o600); err != nil {
			return errors.Wrapf(err, "write %q", name)
		}
	}
	return nil
}

// tdesktopSerializeAuthorization builds the dbiMtpAuthorization record: the
// blockId, the QByteArray-style length prefix that wraps
// Account::serializeMtpAuthorization's own output, and that output itself
// (wide-ID tag, user id, main DC, a single auth key, zero keys-to-destroy).
func tdesktopSerializeAuthorization(userID uint64, dc int, authKey []byte) []byte {
	var inner []byte
	inner = binary.BigEndian.AppendUint64(inner, tdesktopWideIDsTag)
	inner = binary.BigEndian.AppendUint64(inner, userID)
	inner = binary.BigEndian.AppendUint32(inner, uint32(int32(dc)))
	inner = binary.BigEndian.AppendUint32(inner, 1) // one key
	inner = binary.BigEndian.AppendUint32(inner, uint32(int32(dc)))
	inner = append(inner, authKey...)
	inner = binary.BigEndian.AppendUint32(inner, 0) // no keys to destroy

	var payload []byte
	payload = binary.BigEndian.AppendUint32(payload, tdesktopDbiMtpAuthorization)
	payload = binary.BigEndian.AppendUint32(payload, uint32(len(inner)))
	payload = append(payload, inner...)
	return payload
}

// tdesktopPrefixed wraps payload the way tdesktop's EncryptedDescriptor does:
// a little-endian length of the real payload, followed by the payload
// itself, padded with zeroes up to the AES block size (the padding is never
// interpreted back, its only purpose is satisfying the cipher's block size).
func tdesktopPrefixed(payload []byte) []byte {
	out := make([]byte, 4, 4+len(payload))
	binary.LittleEndian.PutUint32(out, uint32(len(payload)))
	out = append(out, payload...)
	if rem := len(out) % aes.BlockSize; rem != 0 {
		out = append(out, make([]byte, aes.BlockSize-rem)...)
	}
	return out
}

// tdesktopAppendArray appends a Qt QByteArray-style big-endian length-prefixed
// blob, matching the outer TDF-file array format read by (*tdesktopFile).readArray.
func tdesktopAppendArray(dst, data []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(data)))
	return append(dst, data...)
}

// tdesktopWriteFile wraps data in a Telegram Desktop storage file container:
// magic, version, data, then an MD5 hash over all of it.
//
// See https://github.com/telegramdesktop/tdesktop/blob/v2.9.8/Telegram/SourceFiles/storage/details/storage_file_utilities.cpp#L473.
func tdesktopWriteFile(data []byte) []byte {
	version := [4]byte{1, 0, 0, 0}

	out := make([]byte, 0, 8+len(data)+md5.Size)
	out = append(out, tdesktopFileMagic[:]...)
	out = append(out, version[:]...)
	out = append(out, data...)

	h := md5.New() // #nosec G401
	_, _ = h.Write(data)
	var packedLength [4]byte
	binary.LittleEndian.PutUint32(packedLength[:], uint32(len(data)))
	_, _ = h.Write(packedLength[:])
	_, _ = h.Write(version[:])
	_, _ = h.Write(tdesktopFileMagic[:])

	return h.Sum(out)
}

// tdesktopLocalKey re-derives Telegram Desktop's local encryption key from a
// passcode and salt.
//
// See https://github.com/telegramdesktop/tdesktop/blob/v2.9.8/Telegram/SourceFiles/storage/details/storage_file_utilities.cpp#L300.
func tdesktopLocalKey(passcode, salt []byte) crypto.Key {
	iters := 1
	if len(passcode) > 0 {
		iters = tdesktopStrongIterations
	}

	h := sha512.New()
	_, _ = h.Write(salt)
	_, _ = h.Write(passcode)
	_, _ = h.Write(salt)

	var r crypto.Key
	key := pbkdf2.Key(h.Sum(nil), salt, iters, len(r), sha512.New)
	copy(r[:], key)
	return r
}

// tdesktopEncryptLocal is the write side of decryptLocal in
// github.com/gotd/td/session/tdesktop, reimplemented here since that package
// only exposes the read direction.
//
// See https://github.com/telegramdesktop/tdesktop/blob/v2.9.8/Telegram/SourceFiles/storage/details/storage_file_utilities.cpp#L584.
func tdesktopEncryptLocal(decrypted []byte, localKey crypto.Key) ([]byte, error) {
	if l := len(decrypted); l%aes.BlockSize != 0 {
		return nil, errors.Errorf("invalid length %d, must be padded to 16", l)
	}

	var msgKey bin.Int128
	h := sha1.Sum(decrypted) // #nosec G401
	copy(msgKey[:], h[:])

	aesKey, aesIV := crypto.OldKeys(localKey, msgKey, crypto.Server)
	cipher, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, errors.Wrap(err, "create cipher")
	}

	encrypted := make([]byte, 16+len(decrypted))
	copy(encrypted, msgKey[:])
	ige.EncryptBlocks(cipher, aesIV[:], encrypted[16:], decrypted)
	return encrypted, nil
}

// tdesktopFileKey reproduces tdesktop's obfuscated on-disk file naming:
// accounts and other storage blocks are named by a nibble-swapped MD5 of
// their logical key, not by that key itself (e.g. "data" always maps to
// "D877F783D5D3EF8C").
func tdesktopFileKey(s string) string {
	hash := md5.Sum([]byte(s)) // #nosec G401
	for i := range hash {
		hash[i] = hash[i]<<4 | hash[i]>>4
	}
	return strings.ToUpper(hex.EncodeToString(hash[:]))[:16]
}
