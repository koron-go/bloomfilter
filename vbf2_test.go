package bloomfilter

import (
	"math/rand"
	"strconv"
	"testing"
)

func TestVBF2Sample100(t *testing.T) {
	vf := NewVBF2(100, 8, 8)
	if vf.nbits != 8 {
		t.Errorf("unexpected vf.nbits: want=8 got=%d", vf.nbits)
	}
	if len(vf.data) != 100 {
		t.Errorf("unexpected len(vf.data): want=100 got=%d", len(vf.data))
	}
	if vf.max != 255 {
		t.Errorf("unexpected vf.max: want=255 got=%d", vf.max)
	}
	samples := make([]uint8, 100)
	for i := range samples {
		samples[i] = uint8(i)
		vf.putData(i, uint8(i))
	}
	for i := range samples {
		x := vf.getData(i)
		if x != samples[i] {
			t.Errorf("data mismatch at %d: want=%02x got=%02x", i, samples[i], x)
			break
		}
	}
}

func checkVBF2(tb testing.TB, m, k, num int, f float64, nbits uint8) {
	tb.Helper()
	mid := int(float64(num) * f)
	vf := NewVBF2(m, k, nbits)
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		vf.Put([]byte(s))
	}

	// check no false negative entries.
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		has := vf.Check([]byte(s), 0)
		if !has {
			tb.Fatalf("false negative: %d", i)
		}
	}

	falsePositive := 0
	for i := mid; i < num; i++ {
		s := strconv.Itoa(i)
		has := vf.Check([]byte(s), 0)
		if has {
			falsePositive++
		}
	}
	errRate := float64(falsePositive) / float64(m) * 100
	//tb.Logf("error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, num, f)
	if errRate > 1 {
		tb.Errorf("too big error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f nbits=%d", errRate, falsePositive, m, k, num, f, nbits)
	}
}

func TestVBF2Basic1(t *testing.T) {
	checkVBF2(t, 1000, 7, 200, 0.1, 1)
	checkVBF2(t, 1000, 7, 200, 0.5, 1)
	checkVBF2(t, 1000, 7, 200, 0.9, 1)
	checkVBF2(t, 1000, 7, 400, 0.1, 1)
	checkVBF2(t, 1000, 7, 700, 0.1, 1)
	checkVBF2(t, 1000, 7, 1000, 0.1, 1)
}

func TestVBF2Basic2(t *testing.T) {
	checkVBF2(t, 1000, 7, 200, 0.1, 2)
	checkVBF2(t, 1000, 7, 200, 0.5, 2)
	checkVBF2(t, 1000, 7, 200, 0.9, 2)
	checkVBF2(t, 1000, 7, 400, 0.1, 2)
	checkVBF2(t, 1000, 7, 700, 0.1, 2)
	checkVBF2(t, 1000, 7, 1000, 0.1, 2)
}

func TestVBF2Basic4(t *testing.T) {
	checkVBF2(t, 1000, 7, 200, 0.1, 4)
	checkVBF2(t, 1000, 7, 200, 0.5, 4)
	checkVBF2(t, 1000, 7, 200, 0.9, 4)
	checkVBF2(t, 1000, 7, 400, 0.1, 4)
	checkVBF2(t, 1000, 7, 700, 0.1, 4)
	checkVBF2(t, 1000, 7, 1000, 0.1, 4)
}

func TestVBF2Basic8(t *testing.T) {
	checkVBF2(t, 1000, 7, 200, 0.1, 8)
	checkVBF2(t, 1000, 7, 200, 0.5, 8)
	checkVBF2(t, 1000, 7, 200, 0.9, 8)
	checkVBF2(t, 1000, 7, 400, 0.1, 8)
	checkVBF2(t, 1000, 7, 700, 0.1, 8)
	checkVBF2(t, 1000, 7, 1000, 0.1, 8)
}

func TestVBF2Subtract(t *testing.T) {
	vf := NewVBF2(2048, 8, 8)
	for i := 1; i <= 255; i++ {
		vf.Subtract(1)
		vf.Put([]byte(strconv.Itoa(i)))
	}
	failure, total := 0, 0
	for bias := 0; bias <= 255; bias++ {
		for i := 1; i <= 255; i++ {
			total++
			want := i > bias
			got := vf.Check([]byte(strconv.Itoa(i)), uint8(bias))
			if got != want {
				if !got {
					t.Fatalf("false negative on: bias=%d i=%d", bias, i)
				}
				// record false positive.
				failure++
			}
		}
	}
	rate := float64(failure) / float64(total) * 100
	//t.Logf("error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	if rate > 1 {
		t.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	}
}

func TestVBF2SubtractOverflow(t *testing.T) {
	vf := NewVBF2(2048, 8, 8)
	tail := 256 + 127
	for i := 1; i <= tail; i++ {
		vf.Subtract(1)
		vf.Put([]byte(strconv.Itoa(i)))
	}
	failure, total := 0, 0
	for bias := 0; bias <= 255; bias++ {
		for i := 1; i <= 511; i++ {
			total++
			want := i > tail-255+bias && i <= tail
			got := vf.Check([]byte(strconv.Itoa(i)), uint8(bias))
			if got != want {
				if !got {
					t.Fatalf("false negative on: bias=%d i=%d", bias, i)
				}
				// record false positive.
				failure++
			}
		}
	}
	rate := float64(failure) / float64(total) * 100
	//t.Logf("error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	if rate > 1 {
		t.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	}
}

func benchmarkPut(b *testing.B, vf *VBF2) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vf.Put([]byte(strconv.Itoa(i)))
	}
}

func BenchmarkVF2_Put_Nbits1(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 7, 1))
}

func BenchmarkVF2_Put_Nbits2(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 7, 2))
}

func BenchmarkVF2_Put_Nbits4(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 7, 4))
}

func BenchmarkVF2_Put_Nbits8(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 7, 8))
}

func benchmarkCheck(b *testing.B, vf *VBF2) {
	b.Helper()
	for i := 0; i < b.N; i++ {
		vf.Put([]byte(strconv.Itoa(i)))
	}
	b.ResetTimer()
	max := b.N * 10
	fail := 0
	for i := 0; i < b.N; i++ {
		x := int(rand.Int31n(int32(max)))
		want := x < b.N
		got := vf.Check([]byte(strconv.Itoa(x)), 0)
		if got != want {
			if !got {
				b.Fatalf("boom!: N=%d x=%d", b.N, x)
			}
			fail++
		}
	}
	rate := float64(fail) / float64(b.N) * 100
	if rate > 1 && b.N > 100 {
		b.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, fail, b.N)
	}
}

func BenchmarkVF2_Check_Nbits1(b *testing.B) {
	benchmarkCheck(b, NewVBF2(b.N*10, 7, 1))
}

func BenchmarkVF2_Check_Nbits2(b *testing.B) {
	benchmarkCheck(b, NewVBF2(b.N*10, 7, 2))
}

func BenchmarkVF2_Check_Nbits4(b *testing.B) {
	benchmarkCheck(b, NewVBF2(b.N*10, 7, 4))
}

func BenchmarkVF2_Check_Nbits8(b *testing.B) {
	benchmarkCheck(b, NewVBF2(b.N*10, 7, 8))
}

func BenchmarkVF2_K1(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 1, 1))
}

func BenchmarkVF2_K2(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 2, 1))
}

func BenchmarkVF2_K3(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 3, 1))
}

func BenchmarkVF2_K4(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 4, 1))
}

func BenchmarkVF2_K5(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 5, 1))
}

func BenchmarkVF2_K6(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 6, 1))
}

func BenchmarkVF2_K7(b *testing.B) {
	benchmarkPut(b, NewVBF2(b.N*10, 7, 1))
}
