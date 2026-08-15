package pipe_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	. "github.com/xtls/xray-core/transport/pipe"
)

func TestPipeReadWrite(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(1024))

	b := buf.New()
	b.WriteString("abcd")
	common.Must(pWriter.WriteMultiBuffer(buf.MultiBuffer{b}))

	b2 := buf.New()
	b2.WriteString("efg")
	common.Must(pWriter.WriteMultiBuffer(buf.MultiBuffer{b2}))

	rb, err := pReader.ReadMultiBuffer()
	common.Must(err)
	if r := cmp.Diff(rb.String(), "abcdefg"); r != "" {
		t.Error(r)
	}
}

func TestPipeInterrupt(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(1024))
	payload := []byte{'a', 'b', 'c', 'd'}
	b := buf.New()
	b.Write(payload)
	common.Must(pWriter.WriteMultiBuffer(buf.MultiBuffer{b}))
	pWriter.Interrupt()

	rb, err := pReader.ReadMultiBuffer()
	if err != io.ErrClosedPipe {
		t.Fatal("expect io.ErrClosePipe, but got ", err)
	}
	if !rb.IsEmpty() {
		t.Fatal("expect empty buffer, but got ", rb.Len())
	}
}

func TestPipeClose(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(1024))
	payload := []byte{'a', 'b', 'c', 'd'}
	b := buf.New()
	common.Must2(b.Write(payload))
	common.Must(pWriter.WriteMultiBuffer(buf.MultiBuffer{b}))
	common.Must(pWriter.Close())

	rb, err := pReader.ReadMultiBuffer()
	common.Must(err)
	if rb.String() != string(payload) {
		t.Fatal("expect content ", string(payload), " but actually ", rb.String())
	}

	rb, err = pReader.ReadMultiBuffer()
	if err != io.EOF {
		t.Fatal("expected EOF, but got ", err)
	}
	if !rb.IsEmpty() {
		t.Fatal("expect empty buffer, but got ", rb.String())
	}
}

func TestPipeLimitZero(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(0))
	bb := buf.New()
	common.Must2(bb.Write([]byte{'a', 'b'}))
	common.Must(pWriter.WriteMultiBuffer(buf.MultiBuffer{bb}))

	var errg errgroup.Group
	errg.Go(func() error {
		b := buf.New()
		b.Write([]byte{'c', 'd'})
		return pWriter.WriteMultiBuffer(buf.MultiBuffer{b})
	})
	errg.Go(func() error {
		time.Sleep(time.Second)

		var container buf.MultiBufferContainer
		if err := buf.Copy(pReader, &container); err != nil {
			return err
		}

		if r := cmp.Diff(container.String(), "abcd"); r != "" {
			return errors.New(r)
		}
		return nil
	})
	errg.Go(func() error {
		time.Sleep(time.Second * 2)
		return pWriter.Close()
	})
	if err := errg.Wait(); err != nil {
		t.Error(err)
	}
}

func TestPipeWriteMultiThread(t *testing.T) {
	pReader, pWriter := New(WithSizeLimit(0))

	var errg errgroup.Group
	for i := 0; i < 10; i++ {
		errg.Go(func() error {
			b := buf.New()
			b.WriteString("abcd")
			return pWriter.WriteMultiBuffer(buf.MultiBuffer{b})
		})
	}
	time.Sleep(time.Millisecond * 100)
	pWriter.Close()
	errg.Wait()

	b, err := pReader.ReadMultiBuffer()
	common.Must(err)
	if r := cmp.Diff(b[0].Bytes(), []byte{'a', 'b', 'c', 'd'}); r != "" {
		t.Error(r)
	}
}

func TestInterfaces(t *testing.T) {
	_ = buf.Reader(new(Reader))
	_ = buf.TimeoutReader(new(Reader))

	_ = common.Interruptible(new(Reader))
	_ = common.Interruptible(new(Writer))
	_ = common.Closable(new(Writer))
}

func TestClearReadCounterWaitsForInFlightCallback(t *testing.T) {
	reader, writer := New(WithoutSizeLimit())
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	detach := reader.SetReadCounter(func(int64) {
		close(callbackStarted)
		<-releaseCallback
	})
	common.Must(writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("payload"))}))

	readDone := make(chan struct{})
	go func() {
		mb, err := reader.ReadMultiBuffer()
		common.Must(err)
		buf.ReleaseMulti(mb)
		close(readDone)
	}()
	<-callbackStarted
	clearDone := make(chan struct{})
	go func() {
		detach()
		close(clearDone)
	}()
	select {
	case <-clearDone:
		t.Fatal("read counter cleared before its in-flight callback completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseCallback)
	<-readDone
	<-clearDone
}

func TestClearWriteCounterWaitsForInFlightCallback(t *testing.T) {
	_, writer := New(WithoutSizeLimit())
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	detach := writer.SetWriteCounter(func(int64) {
		close(callbackStarted)
		<-releaseCallback
	})

	writeDone := make(chan struct{})
	go func() {
		common.Must(writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("payload"))}))
		close(writeDone)
	}()
	<-callbackStarted
	clearDone := make(chan struct{})
	go func() {
		detach()
		close(clearDone)
	}()
	select {
	case <-clearDone:
		t.Fatal("write counter cleared before its in-flight callback completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseCallback)
	<-writeDone
	<-clearDone
}

func TestCounterDetachOwnsItsGeneration(t *testing.T) {
	_, writer := New(WithoutSizeLimit())
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	detachFirst := writer.SetWriteCounter(func(int64) {
		close(firstStarted)
		<-releaseFirst
	})

	firstWriteDone := make(chan struct{})
	go func() {
		common.Must(writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("first"))}))
		close(firstWriteDone)
	}()
	<-firstStarted

	secondCalled := make(chan struct{}, 1)
	detachSecond := writer.SetWriteCounter(func(int64) { secondCalled <- struct{}{} })
	firstDetachDone := make(chan struct{})
	go func() {
		detachFirst()
		close(firstDetachDone)
	}()
	common.Must(writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("second"))}))
	select {
	case <-secondCalled:
	case <-time.After(time.Second):
		t.Fatal("detaching the first counter removed the second generation")
	}
	select {
	case <-firstDetachDone:
		t.Fatal("first counter detached before its own callback completed")
	default:
	}
	close(releaseFirst)
	<-firstWriteDone
	<-firstDetachDone
	detachSecond()
}

func BenchmarkPipeReadWrite(b *testing.B) {
	reader, writer := New(WithoutSizeLimit())
	a := buf.New()
	a.Extend(buf.Size)
	c := buf.MultiBuffer{a}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		common.Must(writer.WriteMultiBuffer(c))
		d, err := reader.ReadMultiBuffer()
		common.Must(err)
		c = d
	}
}
