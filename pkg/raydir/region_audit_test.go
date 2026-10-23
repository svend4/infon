package raydir

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// vectorBrain authors a region the way the pro2 director's offline fallback does:
// the prompt hashed to a SceneVector, rendered by SceneVector.SceneSpec.
type vectorBrain struct{}

func (vectorBrain) Decide(req brain.Request) (brain.Response, error) {
	var st struct {
		Prompt string `json:"prompt"`
	}
	_ = json.Unmarshal(req.State, &st)
	ray, err := json.Marshal(VectorFromPrompt(st.Prompt).SceneSpec())
	return brain.Response{Protocol: brain.Protocol, Kind: "move", Ray: ray}, err
}

// An authored region must cross the wire in one packet (the transport refuses a
// larger one, so it would never arrive) and arrive intact, so a guest builds
// exactly the region the host applied.
func TestAuthoredRegionFitsOnePacket(t *testing.T) {
	limit := network.MaxPacketSize - network.PacketHeaderSize
	check := func(b brain.Brain, prompt string, i int) bool {
		t.Helper()
		host, guest := NewWorld(), NewWorld()
		at := raytrace.Vec3{X: math.Sin(float64(i) * 1.7), Z: 8 + float64(i)*12}
		reg, n, err := host.AuthorRegion(b, prompt, i, at)
		if err != nil {
			t.Fatalf("author %q: %v", prompt, err)
		}
		wire := reg.Encode()
		if _, err := (&network.Packet{Payload: wire}).Encode(); err != nil || len(wire) > limit {
			return false
		}
		got, err := DecodeRegion(wire)
		if err != nil || !reflect.DeepEqual(got, reg) {
			t.Fatalf("%q: round trip mismatch (err=%v)", prompt, err)
		}
		if g := guest.AddRegion(got); g != n || guest.Props() != host.Props() {
			t.Fatalf("%q: guest built %d props, host %d", prompt, g, n)
		}
		return true
	}
	over := 0
	const N = 2000
	for i := 0; i < N; i++ {
		if !check(vectorBrain{}, fmt.Sprintf("prompt %d", i), i) {
			over++
		}
	}
	// a chat line steers the reference director (DirectorPrompt): many keywords
	// stack many objects into one region.
	chatty := "a surreal strange dream: gold night forest, house, crystal rock by the water, quiet and vast, birds float by a pulsing beacon"
	if !check(brain.Local{}, chatty, 7) {
		over++
		t.Errorf("chat-steered reference region exceeds one packet")
	}
	if over > 0 {
		t.Errorf("%d of %d authored regions exceed the %d-byte packet payload", over, N+1, limit)
	}
}

// A region that already fits is untouched by Fit; one that doesn't keeps the
// longest prefix of its objects that fits, and still round-trips.
func TestRegionFit(t *testing.T) {
	small := Region{Index: 3, At: raytrace.Vec3{Z: 20}, Spec: VectorFromPrompt("a calm world").SceneSpec()}
	if !reflect.DeepEqual(small.Fit(), small) {
		t.Error("a fitting region must be returned unchanged")
	}
	big := Region{Index: 9, At: raytrace.Vec3{Z: 116}, Spec: brain.SceneSpec{Name: "Big Place"}}
	for i := 0; i < 300; i++ {
		big.Spec.Objects = append(big.Spec.Objects, brain.ObjSpec{Kind: "sphere", X: float64(i) + 0.123, Y: 1, R: 0.5, Color: [3]float64{0.5, 0.25, 0.125}})
	}
	f := big.Fit()
	wire := f.Encode()
	if len(wire) > MaxRegionBytes {
		t.Fatalf("fitted region is %d bytes, limit %d", len(wire), MaxRegionBytes)
	}
	k := len(f.Spec.Objects)
	if k == 0 || !reflect.DeepEqual(f.Spec.Objects, big.Spec.Objects[:k]) || f.Spec.Name != "Big Place" {
		t.Fatalf("fit should keep a non-empty object prefix and the name, got %d objects name %q", k, f.Spec.Name)
	}
	more := f
	more.Spec.Objects = big.Spec.Objects[:k+1]
	if len(more.Encode()) <= MaxRegionBytes {
		t.Errorf("fit kept %d objects but %d would also fit", k, k+1)
	}
	if got, err := DecodeRegion(wire); err != nil || !reflect.DeepEqual(got, f) {
		t.Errorf("fitted region round trip: err=%v", err)
	}
	// a name that alone overflows the packet is dropped rather than every object.
	huge := Region{Spec: brain.SceneSpec{Name: string(make([]byte, 2000)), Objects: big.Spec.Objects[:2]}}
	if hf := huge.Fit(); len(hf.Encode()) > MaxRegionBytes || len(hf.Spec.Objects) != 2 {
		t.Errorf("overlong name: %d bytes, %d objects", len(hf.Encode()), len(hf.Spec.Objects))
	}
}

// MaxRegionBytes is the transport's real payload limit.
func TestMaxRegionBytesMatchesTransport(t *testing.T) {
	if MaxRegionBytes != network.MaxPacketSize-network.PacketHeaderSize {
		t.Errorf("MaxRegionBytes = %d, transport payload limit = %d", MaxRegionBytes, network.MaxPacketSize-network.PacketHeaderSize)
	}
}

// A SceneVector region a few bytes over the packet limit must keep its lights:
// SceneSpec lists the sun last, and trimming a plain prefix would drop it. Fit
// sheds a non-emissive form instead.
func TestRegionFitKeepsLights(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	found := 0
	for i := 0; i < 20000 && found < 5; i++ {
		var v SceneVector
		for j := range v {
			v[j] = r.Float64()
		}
		reg := Region{Index: i, Spec: v.SceneSpec()}
		if len(reg.Encode()) <= MaxRegionBytes {
			continue
		}
		found++
		f := reg.Fit()
		if n := len(f.Encode()); n > MaxRegionBytes {
			t.Fatalf("vector %v: fitted region is %d bytes, limit %d", v, n, MaxRegionBytes)
		}
		emitters := func(objs []brain.ObjSpec) (n int) {
			for _, o := range objs {
				if o.Emit != ([3]float64{}) {
					n++
				}
			}
			return n
		}
		if got, want := emitters(f.Spec.Objects), emitters(reg.Spec.Objects); got != want {
			t.Errorf("vector %v: fit kept %d of %d emitters", v, got, want)
		}
		if last := f.Spec.Objects[len(f.Spec.Objects)-1]; last.Emit == ([3]float64{}) || last.Kind != "" {
			t.Errorf("vector %v: the sun (last, emissive, no kind) was not kept: %+v", v, last)
		}
		if f.Spec.Objects[0].Kind != "plane" {
			t.Errorf("vector %v: the ground plane was not kept", v)
		}
	}
	if found == 0 {
		t.Skip("no SceneVector region exceeded the packet limit in this sample")
	}
}
