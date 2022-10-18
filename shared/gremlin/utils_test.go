package gremlin

import "testing"

// BenchmarkString/cast-10         		127163260	         9.318 ns/op	       8 B/op	       1 allocs/op
// BenchmarkString/bytesToString-10     1000000000	         0.3172 ns/op	       0 B/op	       0 allocs/op
func BenchmarkString(b *testing.B) {
	data := []byte("qwerty")

	b.Run("cast", func(b *testing.B) {
		b.ReportAllocs()

		var v string
		for i := 0; i < b.N; i++ {
			v = string(data)
		}

		if v != "qwerty" {
			b.Fatal("bad string")
		}
	})

	b.Run("bytesToString", func(b *testing.B) {
		b.ReportAllocs()

		var v string
		for i := 0; i < b.N; i++ {
			v = bytesToString(data)
		}

		if v != "qwerty" {
			b.Fatal("bad string")
		}
	})
}
