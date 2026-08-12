//go:build windows

package platform

import (
	"encoding/base64"
	"errors"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32            = syscall.NewLazyDLL("crypt32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	cryptProtectData   = crypt32.NewProc("CryptProtectData")
	cryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	localFree          = kernel32.NewProc("LocalFree")
)

const cryptProtectUIForbidden = 0x1

func blobFromBytes(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(b)), pbData: &b[0]}
}

func bytesFromBlob(b dataBlob) []byte {
	if b.cbData == 0 || b.pbData == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice(b.pbData, b.cbData)...)
}

// ProtectString encrypts a value with Windows DPAPI for the current user.
// Workbench stores only the resulting ciphertext in state.json.
func ProtectString(value string) (string, error) {
	in := []byte(value)
	ib := blobFromBytes(in)
	var out dataBlob
	r, _, err := cryptProtectData.Call(uintptr(unsafe.Pointer(&ib)), 0, 0, 0, 0, cryptProtectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		if err == syscall.Errno(0) {
			err = errors.New("CryptProtectData failed")
		}
		return "", err
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return base64.RawStdEncoding.EncodeToString(bytesFromBlob(out)), nil
}

func UnprotectString(ciphertext string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	ib := blobFromBytes(raw)
	var out dataBlob
	r, _, callErr := cryptUnprotectData.Call(uintptr(unsafe.Pointer(&ib)), 0, 0, 0, 0, cryptProtectUIForbidden, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		if callErr == syscall.Errno(0) {
			callErr = errors.New("CryptUnprotectData failed")
		}
		return "", callErr
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return string(bytesFromBlob(out)), nil
}
