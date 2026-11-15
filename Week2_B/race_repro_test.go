
package transport

import (
	"sync"
	"testing"
)

// TestRaceBytesSent：模拟未加锁写入，预期检测到 Data Race。
func TestRaceBytesSent(t *testing.T) {
	s := &Stream{}
	start := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			s.bytesSent = true
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			_ = s.BytesSent()
		}
	}()

	close(start)
	wg.Wait()
}

// TestSafeBytesSent：模拟使用同一把锁写入和读取，预期没有 Data Race。
func TestSafeBytesSent(t *testing.T) {
	s := &Stream{}
	start := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			s.mu.Lock()
			s.bytesSent = true
			s.mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			_ = s.BytesSent()
		}
	}()

	close(start)
	wg.Wait()
}

func TestRaceBytesReceived(t *testing.T) {
	s := &Stream{}
	start := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			s.bytesReceived = true
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			_ = s.BytesReceived()
		}
	}()

	close(start)
	wg.Wait()
}

func TestSafeBytesReceived(t *testing.T) {
	s := &Stream{}
	start := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			s.mu.Lock()
			s.bytesReceived = true
			s.mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			_ = s.BytesReceived()
		}
	}()

	close(start)
	wg.Wait()
}
