package utils

import (
	"fmt"
	"testing"
)

// Test random ASCII generation.
//
//	@param t
func TestRandomASCIIString(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(RandomASCIIString(64))
	}
}

// Test RSA public key loading from string.
//
//	@param t
func TestRsaPublicKey(t *testing.T) {
	pemData := []byte("-----BEGIN PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA0go32BWlEaqS48RRG2JAbcY/O3P1StN4k4bahhtPTYD1PUdRvUusCPCSbU/eUaH5tBPCya+ImkWyl72/lNQztSoFP40jbpgNXalTQfyTHfVjxxIRRX1RdJamjZBK8N/x9M6giPYGDXAoyTVv164r0y7an3tblDeBdoXUSwnBZYdqBRMXb+v5F+H6zAYSvOJp2u9He2jXG+l9OKJUlBwikSCde16gIAf4r6LvuENDBdBDLawGYpKtVHeQo+MC5a5gFeEmbWYUtwTReliw1nhILSZlRB00rl3D7RjINVPR4ff9UQcuBC6ePPbt3hIBw2ci3hfuim3EpHNT19+YUfeypBzU+fXbElodPMMS55P39qvduK49SuWAE8oHYgjKIIQEREdZIMQT31VgEymVnN1AXwYR8YoFdcFoEhsyU2GwkBOCzyJBd47B5RChQlamrDusDqekRRMyLkuo/CEGtyZrEd3/dZK9+V5W5lkCm6lUz5npg6pbKjXHWiZh3nbaTpwUrLCiVhkiCzWuLcszpyqb2NHTp2mICj6Vdjy3boBvda5hCYh1dRs81zIjjfsaVHiYUoMyqyEnmvMpFjQ5UJGa8hWHQD4t0RTjGPJqZUiIckj2wqIPu8EaO5n/0vcyS1P9xcfdefqa9d6HXBtnvN3HiYcX3adCoZ7bdFy+smlAEAUCAwEAAQ==\n-----END PUBLIC KEY-----")
	// Load key
	_, err := LoadRsaPublicKey(pemData)
	if err != nil {
		t.Error(err)
	}
}

func TestRsaSignatureVerify(t *testing.T) {
	// Load public key
	pemData := []byte("-----BEGIN PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAq1uKX3ll4G6iU8vBSEEl7/2oBV4bKh0uCrjLEd7y755TDgZ6+k/HlECT3/nBSvQLj2JAczm52Blsx3TtMtgBs6lAZqCC41QAgrHw3hvMWFGCJF3EuIHRqGnMopwxhQ/P7gmTIPCj5Ze70Mp3+oA8gF+wuN6dII2ux1RYwB65vJ3TYuPlGJ85CdUbUwX5k1P+LYfIAg1ITPnLMdX2VyCubhfep7rGb6W/7ADFWa81WOA28wcgdu29ekRKI2cQxZlW/PbcodeocMlpfHO4/cDQWHUYgez19Yp3WkOvAwZHC3Aeyk5NID+ACwdCW2uDKm0AgyZK1uVL78SUqdNcmaJvraoTMT4rAYmCTZoYDnJFB2ZHrVwUHF2KPT63LtUL1gd0NybqHRO0DJMqyQ7qFOZnvla26Izrpzm532WMVWPN/SCGXl4544MM/UoLmJhdBHMZbmI7/LafDp5Jhg3xdNwEnpy7isOVOX8XYnmcDKmvjzB0xd6n/mME+UZHreqbRk8yuLvQFUuoWCNB1MhDsjcrFPgGzqc4RQ7Z6cDG+1w5JEROMH6IMwF0UbQMIsTfxLtQH8utf0Z/bodi67I/XmrpujNDyRg8SXVZR0QrfNhEpNTPjbINCmrRUdkU3IAtru6hgd/Ng16mVTIH8GKl3s8TxissEXyegQGRjFtSx+6RiCsCAwEAAQ==\n-----END PUBLIC KEY-----")
	publicKey, err := LoadRsaPublicKey(pemData)
	if err != nil {
		t.Error(err)
	}
	// Nonce
	nonce := "aifWJOSpwn2IjTom9FKNF6/D56CZRl6NekwbTQd2ixJ4L785/BIkFkGfA3N8jPhduv41pLGvR/tnt/PYV1QSl7x6jx1tHAZeb1aRBas2fymPLChULkzB0xV33FyMjbGWdqthoOfGwquR4XhzDrz6NjSaUxV6XpD9UJFxkVo8JazUNzCrHQRQ9AJWfPn8jFAWActr30Vb/7EHOXOcjxKPukAJscKuQ4Omiw+GuKOgpTeht+JEPgCgGZXT3DyIo7r+qDtzMkm7Z5bb1gmJtA/AEDOlzfIUQRQ9v9Ze5TamlVgGlLUpOMimqskhb1fegvET544oOVEToc5gZ1CubzvqSg=="
	// Signed nonce
	signedNonce := "lf2zVQkMuJyCGkxJGTLfxGyplVKtv6yuFn73HHh+fY9m0sCrSK5Y/kOgKVFAFy4S+SZcEauthf9zXPIFx7ld2HjAo3ALZdf6/ln/nQWXkImy70J30IWPyQP5w4Txb1l19b548vU0Xvdz+eZoRZZeJ7JWCi2RH0RSRlcMoVXJawyHnFnGMKfUhrUI7B0vcmGza9U9FerhfOvXTxeP4QKJ5d1vK+y1Jef1UhnT9i/jdliYMLlaEpvmyshZdX3fCoJo0aLCykjIreL73zSMR+CFUM75FfeBHTZGiMza6LS52UeyC3P2jZl/dO7e/MZ5O9GFms3IYwOgLQLn2w/hyZNSk9lvraxDu0HsNLJ+QQUwc7Dh3qAKGvD3fEDPPavfmyAgHFVZ2fESQIfz4aVN0UZvx3DNHqVw/j7ggb9E04DUpil4GvhDmh2KQuCYqPZygXyOSjilZWUHJbIGH0yn0y2/oY29ZJUkyfflGzjm/4XueQAS8cUyIbzQWXT87slyw9aL9tfnvtYNhiVy9v/G3EjfIRvKG8kouitIlu27edzxsG6LDeMPuRiGpW76lSx84RHH3/+8NyszPL6WoTKYR96i8gstVlYDwQwQtRAH6chj6dvGO7UsGE9ymMZ87hGd6Wr8oxH+nswV8YeamZtEo6D6vbijD7xdDEW4c4XtgbFPjs4="
	// Verify
	err = VerifyRSASignature(publicKey, nonce, signedNonce)
	if err != nil {
		t.Error(err)
	}
}
