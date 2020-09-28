package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

const N = 1000000

var testCases []testCase

func init() {
	rand.Seed(time.Now().UnixNano())
	testCases = genRandomTestCases(N)
}

type testStruct struct {
	a int64
	b int64
	c int64
	d int64
	e int64
	f int64
	g int64
	h int64
	i float64
	j float64
	k float64
	l float64
	m float64
	n float64
	o float64
	p float64
}

func (this *testStruct) String() string {
	return fmt.Sprintf("testStruct:{\n\ta: %v\n\tb: %v\n\tc: %v\n\td: %v\n\te: %v\n\tf: %v\n\tg: %v\n\th: %v\n\ti: %v\n\tj: %v\n\tk: %v\n\tl: %v\n\tm: %v\n\tn: %v\n\to: %v\n\tp: %v\n}\n", this.a, this.b, this.c, this.d, this.e, this.f, this.g, this.h, this.i, this.j, this.k, this.l, this.m, this.n, this.o, this.p)
}

func getRandomTestStruct() testStruct {
	return testStruct{
		a: rand.Int63(),
		b: rand.Int63(),
		c: rand.Int63(),
		d: rand.Int63(),
		e: rand.Int63(),
		f: rand.Int63(),
		g: rand.Int63(),
		h: rand.Int63(),
		i: rand.Float64(),
		j: rand.Float64(),
		k: rand.Float64(),
		l: rand.Float64(),
		m: rand.Float64(),
		n: rand.Float64(),
		o: rand.Float64(),
		p: rand.Float64(),
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func getRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type testCase struct {
	key   string
	value testStruct
}

func genRandomTestCases(n int) []testCase {
	testCases := []testCase{}
	for i := 1; i <= n; i++ {
		testCases = append(testCases, testCase{getRandomString(100), getRandomTestStruct()})
	}
	return testCases
}

func TestSingleThreadSetThenGet(t *testing.T) {
	c := NewCache(10*time.Second, 5*time.Second, N, func(key string, value interface{}) {})
	for _, ca := range testCases {
		err := c.Set(ca.key, ca.value)
		if err != nil {
			t.Error("set key " + ca.key + " failed, message:\n" + err.Error())
			return
		}
	}
	for _, ca := range testCases {
		v, f := c.Get(ca.key)
		if f != nil {
			t.Error("get key " + ca.key + " failed!, message:\n" + f.Error() + "\nshould be:\n" + ca.value.String())
			return
		}
		vs := v.(testStruct)
		if vs != ca.value {
			t.Error("get key " + ca.key + " failed! should be:\n" + ca.value.String() + "but found:\n" + vs.String())
		}
	}
}

func TestSingleThreadSetFailMaxSize(t *testing.T) {
	c := NewCache(10*time.Second, 5*time.Second, N/2, func(key string, value interface{}) {})
	for _, ca := range testCases[:len(testCases)/2] {
		err := c.Set(ca.key, ca.value)
		if err != nil {
			t.Error("set key " + ca.key + " failed, message:\n" + err.Error())
			return
		}
	}
	for _, ca := range testCases[len(testCases)/2:] {
		err := c.Set(ca.key, ca.value)
		if err == nil {
			t.Error("set key " + ca.key + " should failed\n")
			return
		}
	}
}

func TestSingleThreadAddthenGet(t *testing.T) {
	c := NewCache(10*time.Second, 5*time.Second, N, func(key string, value interface{}) {})
	for _, ca := range testCases {
		err := c.Add(ca.key, ca.value)
		if err != nil {
			t.Error("set key " + ca.key + " failed, message:\n" + err.Error())
			return
		}
	}
	for _, ca := range testCases {
		v, f := c.Get(ca.key)
		if f != nil {
			t.Error("get key " + ca.key + " failed!, message:\n" + f.Error() + "\nshould be:\n" + ca.value.String())
			return
		}
		vs := v.(testStruct)
		if vs != ca.value {
			t.Error("get key " + ca.key + " failed! should be:\n" + ca.value.String() + "but found:\n" + vs.String())
		}
	}
}

func TestSingleThreadAddFailMaxSize(t *testing.T) {
	c := NewCache(10*time.Second, 5*time.Second, N/2, func(key string, value interface{}) {})
	for _, ca := range testCases[:len(testCases)/2] {
		err := c.Add(ca.key, ca.value)
		if err != nil {
			t.Error("set key " + ca.key + " failed, message:\n" + err.Error())
			return
		}
	}
	for _, ca := range testCases[len(testCases)/2:] {
		err := c.Add(ca.key, ca.value)
		if err == nil {
			t.Error("set key " + ca.key + " should failed\n")
			return
		}
	}
}

var wg sync.WaitGroup

func BenchmarkMultiThreadSetGetAddMy(b *testing.B) {
	c := NewCache(10*time.Second, 5*time.Second, N, nil)
	b.ResetTimer()
	for _, ca := range testCases {
		c.Set(ca.key, ca.value)
	}
	wg = sync.WaitGroup{}
	for i := 0; i < 160; i++ {
		wg.Add(1)
		go get(c)
	}
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go set(c)
	}
	wg.Wait()
}

func get(c *Cache) {
	defer wg.Done()
	for _, ca := range testCases {
		c.Get(ca.key)
	}
}

func set(c *Cache) {
	defer wg.Done()
	for _, ca := range testCases {
		c.Set(ca.key, ca.value)
	}
}

func BenchmarkMultiThreadSetGetAddGoCache(b *testing.B) {
	c := cache.New(10*time.Second, 5*time.Second)
	b.ResetTimer()
	for _, ca := range testCases {
		c.Set(ca.key, ca.value, 0)
	}
	wg = sync.WaitGroup{}
	for i := 0; i < 160; i++ {
		wg.Add(1)
		go getGoCache(c)
	}
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go setGoCache(c)
	}
	wg.Wait()
}

func getGoCache(c *cache.Cache) {
	defer wg.Done()
	for _, ca := range testCases {
		c.Get(ca.key)
	}
}

func setGoCache(c *cache.Cache) {
	defer wg.Done()
	for _, ca := range testCases {
		c.Set(ca.key, ca.value, 0)
	}
}

func TestExpiration(t *testing.T) {
	c := NewCache(500*time.Microsecond, 250*time.Microsecond, 1000, func(string, interface{}) {})
	c.Add("123", "456")
	<-time.After(1 * time.Second)
	_, f := c.Get("123")
	if f == nil {
		t.Error("should fail")
	}
}
