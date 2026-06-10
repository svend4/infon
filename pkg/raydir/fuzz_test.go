package raydir

import (
	"reflect"
	"testing"
)

// fuzz_test.go fuzzes every wire decoder in the shared-world layer. These parse
// untrusted bytes straight off the network (a peer, or a relay), so the contract
// is: never panic, never read out of bounds, on any input. Where a codec is
// float-free its decode→encode→decode is also asserted stable.

func FuzzDecodeChatMsgs(f *testing.F) {
	f.Add(EncodeChatMsgs([]ChatMsg{{ID: 0x1234_0000_0000_0009, Sender: "alice", Text: "hello"}}))
	f.Add(EncodeChatMsgs(nil))
	f.Add([]byte{0xff, 0xff})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		msgs, err := DecodeChatMsgs(data)
		if err != nil {
			return
		}
		again, err := DecodeChatMsgs(EncodeChatMsgs(msgs))
		if err != nil || !reflect.DeepEqual(msgs, again) {
			t.Fatalf("chat round-trip unstable: %v / %v", err, msgs)
		}
	})
}

func FuzzDecodeVoice(f *testing.F) {
	f.Add(EncodeVoice(0xDEADBEEF, []int16{1, -1, 32767, -32768}))
	f.Add([]byte{0, 0, 0, 1, 0xff, 0xff})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		origin, samples, err := DecodeVoice(data)
		if err != nil {
			return
		}
		o2, s2, err := DecodeVoice(EncodeVoice(origin, samples))
		if err != nil || o2 != origin || !reflect.DeepEqual(samples, s2) {
			t.Fatalf("voice round-trip unstable")
		}
	})
}

func FuzzDecodeAck(f *testing.F) {
	f.Add(EncodeAck([]int{0, 3, 7, 1 << 20}))
	f.Add([]byte{0x00, 0x05})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		have, err := DecodeAck(data)
		if err != nil {
			return
		}
		again, err := DecodeAck(EncodeAck(have))
		if err != nil || !reflect.DeepEqual(have, again) {
			t.Fatalf("ack round-trip unstable")
		}
	})
}

func FuzzDecodePose(f *testing.F) {
	f.Add((Pose{}).Encode())
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodePose(data) // must not panic
	})
}

func FuzzDecodePoseSet(f *testing.F) {
	f.Add(PoseSet{1: {}, 2: {}}.Encode())
	f.Add([]byte{0xff, 0xff, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodePoseSet(data) // must not panic on a bogus count/length
	})
}

func FuzzDecodeRegion(f *testing.F) {
	f.Add(Region{Index: 2, Spec: fallbackSpec()}.Encode())
	f.Add([]byte{0, 0, 0, 1})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeRegion(data) // header + arbitrary JSON tail; must not panic
	})
}

func FuzzDecodeSceneSpec(f *testing.F) {
	f.Add([]byte(`{"objects":[{"kind":"box","s":[1,2,3]}],"light":[1,2,3]}`))
	f.Add([]byte(`{"objects":[{"kind":"mesh","name":"rock"}]}`))
	f.Add([]byte("not json"))
	f.Fuzz(func(t *testing.T, data []byte) {
		// runs json.Unmarshal then BuildScene's sanitiser; must never panic.
		_, _, _ = DecodeSceneSpec(data)
	})
}
