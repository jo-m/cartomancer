package avatar

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeAvatar_Deterministic(t *testing.T) {
	seed := "test-seed-abc123"
	a := MakeAvatar(seed, false)
	b := MakeAvatar(seed, false)
	require.Equal(t, a, b, "same seed must produce identical output")
}

func TestMakeAvatar_DifferentSeeds(t *testing.T) {
	a := MakeAvatar("seed-one", false)
	b := MakeAvatar("seed-two", false)
	require.NotEqual(t, a, b, "different seeds should produce different avatars")
}

func TestMakeAvatar_ValidSVG(t *testing.T) {
	svg := MakeAvatar("hello", false)
	require.True(t, strings.HasPrefix(svg, `<svg xmlns="http://www.w3.org/2000/svg"`))
	require.True(t, strings.HasSuffix(svg, `</svg>`))
	require.Contains(t, svg, `viewBox="0 0 20 20"`)
	require.Contains(t, svg, `width="20"`)
	require.Contains(t, svg, `height="20"`)
}

func TestMakeAvatar_ContainsFace(t *testing.T) {
	// Every avatar must have at least a face shape element.
	seeds := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	for _, seed := range seeds {
		svg := MakeAvatar(seed, false)
		hasFace := strings.Contains(svg, "<circle") ||
			strings.Contains(svg, "<ellipse") ||
			strings.Contains(svg, "<rect")
		require.True(t, hasFace, "avatar for seed %q must contain a face shape", seed)
	}
}

func TestMakeAvatar_EmptySeed(t *testing.T) {
	svg := MakeAvatar("", false)
	require.True(t, strings.HasPrefix(svg, `<svg`))
	require.True(t, strings.HasSuffix(svg, `</svg>`))
}

func TestMakeAvatar_Variety(t *testing.T) {
	// Generate many avatars and verify we get meaningful variety.
	seen := make(map[string]bool)
	for i := range 100 {
		svg := MakeAvatar(strings.Repeat("x", i+1), false)
		seen[svg] = true
	}
	require.Greater(t, len(seen), 20, "100 different seeds should produce at least 20 unique avatars")
}

func TestMakeAvatar_Aura(t *testing.T) {
	withAura := MakeAvatar("test", true)
	withoutAura := MakeAvatar("test", false)
	require.Contains(t, withAura, "aura")
	require.NotContains(t, withoutAura, "aura")
	require.NotEqual(t, withAura, withoutAura)
}

func TestRNG_Deterministic(t *testing.T) {
	a := newRNG("seed")
	b := newRNG("seed")
	for range 50 {
		require.Equal(t, a.next(100), b.next(100))
	}
}

func TestRNG_ExhaustsAndRehashes(t *testing.T) {
	r := newRNG("seed")
	// SHA-256 produces 32 bytes; drawing more than 32 values forces a re-hash.
	results := make([]int, 64)
	for i := range results {
		results[i] = r.next(256)
	}
	// Just verify no panic and values are in range.
	for i, v := range results {
		require.GreaterOrEqual(t, v, 0, "value at index %d", i)
		require.Less(t, v, 256, "value at index %d", i)
	}
}

func TestRNG_NextZero(t *testing.T) {
	r := newRNG("seed")
	require.Equal(t, 0, r.next(0))
	require.Equal(t, 0, r.next(-1))
}
