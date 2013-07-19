package main

import (
	"os"
	"fmt"
	"bytes"
	"bufio"
	"strings"
	"encoding/hex"
	"github.com/piotrnar/gocoin/btc"
)


// get public key in bitcoin protocol format, from the give private key
func priv2pub(curv *btc.BitCurve, priv_key []byte, compressed bool) (res []byte) {
	x, y := curv.ScalarBaseMult(priv_key)
	xd := x.Bytes()

	if len(xd)>32 {
		println("x is too long:", len(xd))
		os.Exit(2)
	}

	if !compressed {
		yd := y.Bytes()
		if len(yd)>32 {
			println("y is too long:", len(yd))
			os.Exit(2)
		}

		res = make([]byte, 65)
		res[0] = 4
		copy(res[1+32-len(xd):33], xd)
		copy(res[33+32-len(yd):65], yd)
	} else {
		res = make([]byte, 33)
		res[0] = 2+byte(y.Bit(0)) // 02 for even Y values, 03 for odd..
		copy(res[1+32-len(xd):33], xd)
	}

	return
}

func load_others() {
	f, e := os.Open(RawKeysFilename)
	if e == nil {
		defer f.Close()
		td := bufio.NewReader(f)
		for {
			li, _, _ := td.ReadLine()
			if li == nil {
				break
			}
			pk := strings.SplitN(strings.Trim(string(li), " "), " ", 2)
			if pk[0][0]=='#' {
				continue // Just a comment-line
			}

			pkb := btc.Decodeb58(pk[0])

			if pkb == nil {
				println("Decodeb58 failed:", pk[0][:6])
				continue
			}

			if len(pkb)!=37 && len(pkb)!=38 {
				println(pk[0][:6], "has wrong key", len(pkb))
				println(hex.EncodeToString(pkb))
				continue
			}

			if pkb[0]!=privver {
				println(pk[0][:6], "has version", pkb[0], "while we expect", privver)
				if pkb[0]==0xef {
					fmt.Println("You probably meant testnet, so use -t switch")
					os.Exit(0)
				} else {
					continue
				}
			}

			var sh [32]byte
			var compr bool

			if len(pkb)==37 {
				// compressed key
				sh = btc.Sha2Sum(pkb[0:33])
				if !bytes.Equal(sh[:4], pkb[33:37]) {
					println(pk[0][:6], "checksum error")
					continue
				}
				compr = false
			} else {
				if pkb[33]!=1 {
					println(pk[0][:6], "a key of length 38 bytes must be compressed")
					continue
				}

				sh = btc.Sha2Sum(pkb[0:34])
				if !bytes.Equal(sh[:4], pkb[34:38]) {
					println(pk[0][:6], "checksum error")
					continue
				}
				compr = true
			}

			var key [32]byte
			copy(key[:], pkb[1:33])
			priv_keys = append(priv_keys, key)
			publ_addrs = append(publ_addrs,
				btc.NewAddrFromPubkey(btc.PublicFromPrivate(key[:], compr), verbyte))
			if len(pk)>1 {
				labels = append(labels, pk[1])
			} else {
				labels = append(labels, fmt.Sprint("Other ", len(priv_keys)))
			}
		}
		if *verbose {
			fmt.Println(len(priv_keys), "keys imported from", RawKeysFilename)
		}
	} else {
		if *verbose {
			fmt.Println("You can also have some dumped (b58 encoded) priv keys in file", RawKeysFilename)
		}
	}
}


// Get the secret seed and generate "*keycnt" key pairs (both private and public)
func make_wallet() {
	if *testnet {
		verbyte = 0x6f
		privver = 0xef
	} else {
		// verbyte is be zero by definition
		privver = 0x80
	}
	load_others()

	pass := getpass()
	seed_key := btc.Sha2Sum([]byte(pass))
	if pass!="" {
		if *verbose {
			fmt.Println("Generating", *keycnt, "keys, version", verbyte,"...")
		}
		for i:=uint(0); i < *keycnt; i++ {
			seed_key = btc.Sha2Sum(seed_key[:])
			priv_keys = append(priv_keys, seed_key)
			publ_addrs = append(publ_addrs,
				btc.NewAddrFromPubkey(btc.PublicFromPrivate(seed_key[:], !*uncompressed), verbyte))
			labels = append(labels, fmt.Sprint("Auto ", i+1))
		}
		if *verbose {
			fmt.Println("Private keys re-generated")
		}
	}
}
