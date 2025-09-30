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
