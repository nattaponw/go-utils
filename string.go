package utils

import (
	"crypto/md5"  // #nosec
	"crypto/sha1" // #nosec
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sigurn/crc16"
	"golang.org/x/crypto/bcrypt"
)

var crc16Table *crc16.Table

func init() {
	crc16Table = crc16.MakeTable(crc16.CRC16_CCITT_FALSE)
}

// Constants representing various Unicode characters
const (
	zeroWidthSpace        = '\u200B'
	zeroWidthNoBreakSpace = '\uFEFF'
	wordJoiner            = '\u2060'
	zeroWidthJoiner       = '\u200D'
	leftToRightMark       = '\u200E'
	rightToLeftMark       = '\u200F'
	noBreakingSpace       = '\u00A0'
	empty                 = ""
)

var zeroWithReplacer = strings.NewReplacer(
	string(zeroWidthSpace), empty,
	string(zeroWidthNoBreakSpace), empty,
	string(wordJoiner), empty,
	string(zeroWidthJoiner), empty,
	string(leftToRightMark), empty,
	string(rightToLeftMark), empty,
	string(noBreakingSpace), empty,
)

// EMVCoQRInfo for QR string
type EMVCoQRInfo struct {
	Format          string
	MerchantAccount string
	Amount          string
	PhoneNumber     string
	CountryCode     string
	Crc             string
	CurrencyISO4217 string
	BillerID        string
	Ref1            string
	Ref2            string
	Ref3            string
}

// String utility instance
var String StringUtil

// StringUtil provides utility functions for manipulating strings
type StringUtil struct{}

// RemoveDuplicateSpaces removes duplicate spaces from the input string
func (StringUtil) RemoveDuplicateSpaces(s string) string {
	return strings.Join(strings.Fields(zeroWithReplacer.Replace(s)), " ")
}

// RemoveAllSpaces removes all spaces from the input string
func (StringUtil) RemoveAllSpaces(s string) string {
	return strings.Join(strings.Fields(zeroWithReplacer.Replace(s)), "")
}

// UUID generates a new UUID string
func (StringUtil) UUID() string {
	return uuid.NewString()
}

// MD5 generates an MD5 hash for the input string
func (StringUtil) MD5(s string) string {
	hash := md5.Sum([]byte(s)) // #nosec
	return hex.EncodeToString(hash[:])
}

// SHA1 generates a SHA1 hash for the input string
func (StringUtil) SHA1(s string) string {
	hash := sha1.Sum([]byte(s)) // #nosec
	return hex.EncodeToString(hash[:])
}

// SHA256 generates a SHA256 hash for the input string
func (StringUtil) SHA256(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// HashPassword takes a plaintext password and returns its bcrypt hash
func (StringUtil) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks if the provided plain text password matches the existing bcrypt hash
func (StringUtil) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// HashCrc32 generates a CRC32 hash for the input string
func (StringUtil) HashCrc32(s string) string {
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(s)))
}

// Parse EMVCoQR string to struct
func (StringUtil) ParseEMVCoQRString(qrString string) (EMVCoQRInfo, error) {
	if err := ValidateEMVCoQRString(qrString); err != nil {
		return EMVCoQRInfo{}, err
	}
	result := EMVCoQRInfo{}
	index := 0
	for index < len(qrString) {
		if index+4 > len(qrString) {
			return EMVCoQRInfo{}, fmt.Errorf("invalid qr structure")
		}
		id := qrString[index : index+2]
		length, err := strconv.Atoi(qrString[index+2 : index+4])
		if err != nil {
			return EMVCoQRInfo{}, fmt.Errorf("invalid qr structure")
		}
		if index+4+length > len(qrString) {
			return EMVCoQRInfo{}, fmt.Errorf("invalid specified qr string length")
		}
		value := qrString[index+4 : index+4+length]
		switch id {
		case "01":
			result.Format = value
		case "29":
			prefixPhoneIndex := strings.Index(value, "011300")
			result.MerchantAccount = value
			if prefixPhoneIndex != -1 {
				result.PhoneNumber = value[prefixPhoneIndex+6:]
			}
		case "30":
			result.MerchantAccount = value
			index2 := 0
			for index2 < len(value) {
				if index2+4 > len(value) {
					return EMVCoQRInfo{}, fmt.Errorf("invalid qr structure")
				}
				id2 := value[index2 : index2+2]
				length2, err := strconv.Atoi(value[index2+2 : index2+4])
				if err != nil {
					return EMVCoQRInfo{}, fmt.Errorf("invalid qr structure")
				}
				if index2+4+length2 > len(value) {
					return EMVCoQRInfo{}, fmt.Errorf("invalid specified qr string length")
				}
				value2 := value[index2+4 : index2+4+length2]
				switch id2 {
				case "01":
					result.BillerID = value2
				case "02":
					result.Ref1 = value2
				case "03":
					result.Ref2 = value2
				}
				index2 += 4 + length2
			}
		case "54":
			result.Amount = value
		case "58":
			result.CountryCode = value
		case "62":
			if len(value) > 4 {
				result.Ref3 = value[4:]
			}
		case "63":
			result.Crc = value
		case "53":
			result.CurrencyISO4217 = value
		}
		index += 4 + length
	}
	return result, nil
}

// ValidateEMVCoQRString validates the EMVCoQR string
func ValidateEMVCoQRString(qrString string) error {
	if len(qrString) < 14 {
		return fmt.Errorf("invalid specified qr string length")
	}
	data := []byte(qrString[:len(qrString)-4])
	crc := crc16.Checksum(data, crc16Table)
	calculatedCRC := fmt.Sprintf("%04X", int(crc))
	expectedCRC := qrString[len(qrString)-4:]
	if calculatedCRC != expectedCRC {
		return fmt.Errorf("qr checksum is incorrect")
	}
	return nil
}
