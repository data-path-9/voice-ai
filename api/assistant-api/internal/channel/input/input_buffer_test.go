package internal_channel_input

import "testing"

func TestBytesInputBuffer_DrainIfReady_HoldsUntilThreshold(t *testing.T) {
	b := NewBytesInputBuffer(0)
	b.Write([]byte{1, 2, 3})
	if out, ok := b.DrainIfReady(4); ok || out != nil {
		t.Fatal("expected no drain below threshold")
	}
	if got := b.Len(); got != 3 {
		t.Fatalf("expected len=3 got=%d", got)
	}
}

func TestBytesInputBuffer_DrainIfReady_DrainsAllWhenReady(t *testing.T) {
	b := NewBytesInputBuffer(0)
	b.Write([]byte{1, 2, 3, 4, 5})
	out, ok := b.DrainIfReady(4)
	if !ok {
		t.Fatal("expected drain when threshold reached")
	}
	if len(out) != 5 {
		t.Fatalf("expected full drain length 5 got=%d", len(out))
	}
	if b.Len() != 0 {
		t.Fatalf("expected empty buffer after drain")
	}
}

func TestBytesInputBuffer_Clear(t *testing.T) {
	b := NewBytesInputBuffer(0)
	b.Write([]byte{1, 2, 3})
	b.Clear()
	if b.Len() != 0 {
		t.Fatalf("expected empty after clear")
	}
}
