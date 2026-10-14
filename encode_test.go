package htpasswd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeRoundTrip(t *testing.T) {
	algorithms := []Algorithm{
		AlgorithmAPR1,
		AlgorithmBcrypt,
		AlgorithmSHA,
		AlgorithmPlain,
		AlgorithmCryptSHA256,
		AlgorithmCryptSHA512,
	}

	for _, algorithm := range algorithms {
		t.Run(string(algorithm), func(t *testing.T) {
			encoded, err := Encode("correct horse battery staple", EncodeOptions{Algorithm: algorithm, Cost: 4})
			require.NoError(t, err)

			parsed, err := NewFromReader(strings.NewReader("alice:"+encoded+"\n"), DefaultSystems, nil)
			require.NoError(t, err)
			require.True(t, parsed.Match("alice", "correct horse battery staple"))
			require.False(t, parsed.Match("alice", "wrong"))
		})
	}
}
