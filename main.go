package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/pastelnetwork/go-raptorq/pkg/defaults"
	"golang.org/x/crypto/sha3"
)

func getFileContentHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer f.Close()
	hash := sha3.New256()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func encode() (uint64, uint32, error) {
	src, err := ioutil.ReadFile("input.png")
	if err != nil {
		fmt.Println("File reading error", err)
		return 0, 0, err
	}
	enc, err := defaults.NewEncoder(src, 50*1024, 50*1024, (4 * 1024 * 1024), 128)
	if err != nil {
		fmt.Println("Error creating Encoder", err)
		return 0, 0, err
	}
	if enc == nil {
		fmt.Println("Cannot create Encoder")
		return 0, 0, err
	}
	sbNumber := enc.NumSourceBlocks()
	subNumber := enc.NumSubBlocks()
	fmt.Printf("Encoder: NumSourceBlocks - %d; NumSubBlocks - %d\n", sbNumber, subNumber)
	var sbn uint8
	for sbn = 0; sbn < sbNumber; sbn++ {
		smbNumber := enc.NumSourceSymbols(sbn)
		smbSize := enc.SymbolSize()
		fmt.Printf("Encoder: Source block %d: NumSourceSymbols - %d; SymbolSize - %d\n", sbn, smbNumber, smbSize)
		minSym := enc.MinSymbols(sbn)
		maxSym := enc.MaxSymbols(sbn)
		fmt.Printf("Encoder: Source block %d: MinSymbols - %d; MaxSymbols - %d\n", sbn, minSym, maxSym)
		symbolsCount := uint32(minSym * 5)
		if symbolsCount > maxSym {
			symbolsCount = maxSym
		}
		if _, err := os.Stat("symbols"); os.IsNotExist(err) {
			if err = os.MkdirAll("symbols", 0770); err != nil {
				return 0, 0, err
			}
		}
		fmt.Printf("Encoder: Source block %d: Encoding - %d source bytes; into %d Symbols; %d bytes each\n", sbn, enc.TransferLength(), symbolsCount, smbSize)
		symbol := make([]byte, smbSize)
		var esi uint32
		for esi = 0; esi < symbolsCount; esi++ {
			w, err := enc.Encode(sbn, esi, symbol)
			if err != nil {
				fmt.Printf("Error getting symbol %d-%d - %s\n", sbn, esi, err)
				return 0, 0, err
			}
			if w != 0 {
				f, _ := os.Create(fmt.Sprintf("symbols/%d-%d", sbn, esi))
				f.Write(symbol)
				defer f.Close()
			}

			fileHash, _ := getFileContentHash(fmt.Sprintf("symbols/%d-%d", sbn, esi))
			fmt.Printf("%v\n", fileHash)
		}
	}
	cOTI := enc.CommonOTI()
	ssOTI := enc.SchemeSpecificOTI()
	fmt.Printf("Encoder: CommonOTI - %d; SchemeSpecificOTI - %d\n", cOTI, ssOTI)
	defer enc.Close()
	return cOTI, ssOTI, nil
}

func decode(cOTI uint64, ssOTI uint32) {
	dec, err := defaults.NewDecoder(cOTI, ssOTI)
	if err != nil {
		fmt.Println("Error creating Decoder", err)
		return
	}
	if dec == nil {
		fmt.Println("Cannot create Decoder")
		return
	}
	sourceSize := dec.TransferLength()
	fmt.Printf("Decoder: Source file should be %d bytes\n", sourceSize)
	sbNumberDec := dec.NumSourceBlocks()
	subNumberDec := dec.NumSubBlocks()
	fmt.Printf("Decoder: Should be %d SourceBlocks and %d SubBlocks\n", sbNumberDec, subNumberDec)
	files, err := ioutil.ReadDir("./symbols")
	if err != nil {
		fmt.Printf("Cannot list files in directory - %s", "symbols")
		return
	}
	for _, f := range files {
		fmt.Printf("Reading file - %s\n", f.Name())
		t := strings.Split(f.Name(), "-")
		if len(t) == 2 {
			b, err := strconv.ParseUint(t[0], 10, 8)
			if err != nil {
				fmt.Printf("Wrong file name - %s", f.Name())
				continue
			}
			s, err := strconv.ParseUint(t[1], 10, 8)
			if err != nil {
				fmt.Printf("Wrong file name - %s", f.Name())
				continue
			}
			symFileName := fmt.Sprintf("symbols/%s", f.Name())
			symBlock, err := ioutil.ReadFile(symFileName)
			if err != nil {
				fmt.Printf("File reading error %s; file - %s", err, f.Name())
				return
			}
			dec.Decode(uint8(b), uint32(s), symBlock)
			if dec.IsSourceObjectReady() {
				source := make([]byte, sourceSize)
				bytes, err := dec.SourceObject(source)
				if err != nil {
					fmt.Println("Error getting Decoded source", err)
					return
				}
				if bytes == 0 {
					fmt.Println("Error getting Decoded source - get 0 bytes")
					return
				}
				sourceFile, _ := os.Create("output.png")
				sourceFile.Write(source)
				sourceFile.Close()
				break
			}
		}
	}
	err = dec.Close()
	if err != nil {
		fmt.Println("Error closing Decoder", err)
		return
	}
}

func main() {
	counter := 1
	for {
		fmt.Printf("\nstarting iteration %v\n", counter)
		cOTI, ssOTI, _ := encode()
		decode(cOTI, ssOTI)
		counter++
	}
}
