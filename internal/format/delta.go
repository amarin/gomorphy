package format

import (
	"encoding/binary"
	"fmt"
)

func AppendDelta(dst []byte, vals []uint64) []byte {
	dst = binary.AppendUvarint(dst, uint64(len(vals)))

	var prev uint64
	for i, v := range vals {
		if i == 0 {
			dst = binary.AppendUvarint(dst, v)
			prev = v

			continue
		}

		dst = binary.AppendUvarint(dst, zigzagEncode(int64(v-prev)))
		prev = v
	}

	return dst
}

func ParseDelta(data []byte) ([]uint64, error) {
	count, n := binary.Uvarint(data)
	if n <= 0 {
		return nil, ErrMalformedDelta
	}

	data = data[n:]
	vals := make([]uint64, 0, min(count, 1<<20))

	var prev uint64

	for i := range count {
		x, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, wrap(ErrMalformedDelta, fmt.Sprintf("item %d", i))
		}

		data = data[n:]

		if i == 0 {
			prev = x
			vals = append(vals, x)

			continue
		}

		prev += uint64(zigzagDecode(x))
		vals = append(vals, prev)
	}

	return vals, nil
}

func AppendDelta32(dst []byte, vals []uint32) []byte {
	src := make([]uint64, len(vals))
	for i, v := range vals {
		src[i] = uint64(v)
	}

	return AppendDelta(dst, src)
}

func ParseDelta32(data []byte) ([]uint32, error) {
	vals, err := ParseDelta(data)
	if err != nil {
		return nil, err
	}

	out := make([]uint32, len(vals))
	for i, v := range vals {
		if v > uint64(^uint32(0)) {
			return nil, wrap(ErrMalformedDelta, fmt.Sprintf("value %d overflows uint32 at %d", v, i))
		}

		out[i] = uint32(v)
	}

	return out, nil
}

func zigzagEncode(d int64) uint64 {
	return (uint64(d) << 1) ^ uint64(d>>63)
}

func zigzagDecode(x uint64) int64 {
	return int64(x>>1) ^ -(int64(x) & 1)
}
