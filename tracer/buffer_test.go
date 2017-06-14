package tracer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//func NewSpan(name, service, resource string, spanID, traceID, parentID uint64, tracer *Tracer) *Span {

const (
	testBufferTimeout = time.Second // shorter than go test timeout to avoid long CI builds
	testInitSize      = 2
	testMaxSize       = 5
)

func TestTraceBufferPushOne(t *testing.T) {
	assert := assert.New(t)

	traceChan := make(chan []*Span)
	errChan := make(chan error)

	buffer := newTraceBuffer(traceChan, errChan, testInitSize, testMaxSize)
	assert.NotNil(buffer)
	assert.Len(buffer.spans, 0)

	traceID := NextSpanID()
	root := NewSpan("name1", "a-service", "a-resource", traceID, traceID, 0, nil)
	root.buffer = buffer

	buffer.Push(root)
	assert.Len(buffer.spans, 1, "there is one span in the buffer")
	assert.Equal(root, buffer.spans[0], "the span is the one pushed before")

	go root.Finish() // use a goroutine as channel has size 0

	select {
	case trace := <-traceChan:
		assert.Len(trace, 1, "there was trace in the channel")
		assert.Equal(root, trace[0], "the trace in the channel is the one pushed before")
		assert.Equal(0, buffer.Len(), "no more spans in the buffer")
	case err := <-errChan:
		assert.Fail("unexpected error:", err.Error())
		t.Logf("buffer: %v", buffer)
	case <-time.After(testBufferTimeout):
		assert.Fail("timeout")
		t.Logf("buffer: %v", buffer)
	}
}

func TestTraceBufferPushNoFinish(t *testing.T) {
	assert := assert.New(t)

	traceChan := make(chan []*Span)
	errChan := make(chan error)

	buffer := newTraceBuffer(traceChan, errChan, testInitSize, testMaxSize)
	assert.NotNil(buffer)
	assert.Len(buffer.spans, 0)

	traceID := NextSpanID()
	root := NewSpan("name1", "a-service", "a-resource", traceID, traceID, 0, nil)
	root.buffer = buffer

	buffer.Push(root)
	assert.Len(buffer.spans, 1, "there is one span in the buffer")
	assert.Equal(root, buffer.spans[0], "the span is the one pushed before")

	select {
	case <-traceChan:
		assert.Fail("span was not finished, should not be flushed")
		t.Logf("buffer: %v", buffer)
	case err := <-errChan:
		assert.Fail("unexpected error:", err.Error())
		t.Logf("buffer: %v", buffer)
	case <-time.After(testBufferTimeout):
		assert.Len(buffer.spans, 1, "there is still one span in the buffer")
	}
}

func TestTraceBufferPushSeveral(t *testing.T) {
	assert := assert.New(t)

	traceChan := make(chan []*Span)
	errChan := make(chan error)

	buffer := newTraceBuffer(traceChan, errChan, testInitSize, testMaxSize)
	assert.NotNil(buffer)
	assert.Len(buffer.spans, 0)

	traceID := NextSpanID()
	root := NewSpan("name1", "a-service", "a-resource", traceID, traceID, 0, nil)
	span2 := NewSpan("name2", "a-service", "a-resource", NextSpanID(), traceID, root.SpanID, nil)
	span3 := NewSpan("name3", "a-service", "a-resource", NextSpanID(), traceID, root.SpanID, nil)
	span3a := NewSpan("name3", "a-service", "a-resource", NextSpanID(), traceID, span3.SpanID, nil)

	spans := []*Span{root, span2, span3, span3a}

	for i, span := range spans {
		span.buffer = buffer
		buffer.Push(span)
		assert.Len(buffer.spans, i+1, "there is one more span in the buffer")
		assert.Equal(span, buffer.spans[i], "the span is the one pushed before")
	}

	for _, span := range spans {
		go span.Finish() // use a goroutine as channel has size 0
	}

	select {
	case trace := <-traceChan:
		assert.Len(trace, 4, "there was one trace with the right number of spans in the channel")
		for _, span := range spans {
			assert.Contains(trace, span, "the trace contains the spans")
		}
	case err := <-errChan:
		assert.Fail("unexpected error:", err.Error())
	case <-time.After(testBufferTimeout):
		assert.Fail("timeout")
	}
}
