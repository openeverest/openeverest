package httputil

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		client := NewClient()
		assert.Equal(t, time.Duration(0), client.Timeout)

		tr, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		assert.False(t, tr.DisableKeepAlives)
	})

	t.Run("with options", func(t *testing.T) {
		client := NewClient(
			WithTimeout(10*time.Second),
			WithInsecureSkipVerify(true),
			WithTransient(),
		)

		assert.Equal(t, 10*time.Second, client.Timeout)

		tr, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		assert.True(t, tr.DisableKeepAlives)
		require.NotNil(t, tr.TLSClientConfig)
		assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("isolation", func(t *testing.T) {
		c1 := NewClient(WithInsecureSkipVerify(true))
		c2 := NewClient(WithInsecureSkipVerify(false))

		tr1 := c1.Transport.(*http.Transport)
		tr2 := c2.Transport.(*http.Transport)

		assert.NotSame(t, tr1, tr2)
		require.NotNil(t, tr1.TLSClientConfig)
		assert.True(t, tr1.TLSClientConfig.InsecureSkipVerify)

		if tr2.TLSClientConfig != nil {
			assert.False(t, tr2.TLSClientConfig.InsecureSkipVerify)
		}
	})
}
