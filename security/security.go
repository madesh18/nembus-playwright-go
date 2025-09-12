package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"
)

const Enc_key = "c4798dafcc81b93ba54030fab24415f0"

func Encrypt(stringToEncrypt string, keyString string) (string, error) {

	//Since the key is in string, we need to convert decode it to bytes
	key, _ := hex.DecodeString(keyString)
	plaintext := []byte(stringToEncrypt)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%x", ciphertext), nil
}

func Decrypt(encryptedString string, keyString string) (string, error) {

	key, _ := hex.DecodeString(keyString)
	enc, _ := hex.DecodeString(encryptedString)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", plaintext), nil
}

func GetKey() (string, error) {
	var key string

	if runtime.GOOS == "windows" {
		uuid, err := GetWindowsUUID()
		if err != nil {
			log.Println("E! unable to get the UUID")
			return "", err
		}
		if len(uuid) != 32 {
			log.Println("E! UUID length is not equal to 32. UUID Value - ", uuid)
		}
		key = uuid
	} else {
		var serialNumber string
		var errVAl error
		if runtime.GOOS == "linux" {
			serialNumber, errVAl = GetMachineId()
		} else {
			serialNumber, errVAl = GetSerialNumber()
		}

		if errVAl != nil {
			log.Println("E! Unable to fetch the key", errVAl)
			return "", errVAl
		}

		reg, err := regexp.Compile("[^a-zA-Z0-9]+")
		if err != nil {
			log.Println("E! error while creating regex pattern")
		}

		//Safer side remove special character and append Some Random String.
		// Never touch this code. Else agent wont work
		toRetKey := hex.EncodeToString([]byte(reg.ReplaceAllString(serialNumber+"ZCBMQRTVFDGMPOTR", "")))

		//log.Println("toRetKey len is",len([]byte(toRetKey)))

		key = SplitString(toRetKey, 32)[0]
	}

	return key, nil
}

func GetMachineId() (string, error) {
	buf := &bytes.Buffer{}
	err := run(buf, os.Stderr, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err != nil {
		return "", err
	}
	id, err := extractID(buf.String())
	if err != nil {
		return "", err
	}
	return trim(id), nil
}

func run(stdout, stderr io.Writer, cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}

func trim(s string) string {
	return strings.TrimSpace(strings.Trim(s, "\n"))
}

func extractID(lines string) (string, error) {
	for _, line := range strings.Split(lines, "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.SplitAfter(line, `" = "`)
			if len(parts) == 2 {
				return strings.TrimRight(parts[1], `"`), nil
			}
		}
	}
	return "", fmt.Errorf("Failed to extract 'IOPlatformUUID' value from `ioreg` output.\n%s", lines)
}

func SplitString(longString string, maxLen int) []string {
	splits := []string{}

	var l, r int
	for l, r = 0, maxLen; r < len(longString); l, r = r, r+maxLen {
		for !utf8.RuneStart(longString[r]) {
			r--
		}
		splits = append(splits, longString[l:r])
	}
	splits = append(splits, longString[l:])
	return splits
}
