package storage_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/amarin/gomorphy/pkg/storage"

	"github.com/stretchr/testify/require"
)

func TestRunes_WriteTo(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Runes
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", storage.Runes{}, "00", 1, false},
		{"latin_a", storage.Runes("a"), "0100000061", 5, false},
		{"cyrillic_ab", storage.Runes("АБ"), "020000041000000411", 9, false},
		{"cyrillic_ne", storage.Runes("не"), "020000043d00000435", 9, false},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			wantData, err := hex.DecodeString(tt.wantW)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			gotN, writeErr := tt.grammemesSet.WriteTo(buf)
			require.Equal(t, tt.wantErr, writeErr != nil)
			if writeErr == nil {
				require.Equal(t, tt.wantN, gotN)
				require.Equal(t, wantData, buf.Bytes(), "want 0x%v,\n got 0x%v", tt.wantW, hex.EncodeToString(buf.Bytes()))
			}
		})
	}
}

func TestRunes_ReadFrom(t *testing.T) { //nolint:paralleltest
	for _, tt := range []struct { //nolint:paralleltest
		name         string
		grammemesSet storage.Runes
		wantW        string
		wantN        int64
		wantErr      bool
	}{
		{"empty", storage.Runes{}, "00", 0, false},
		{"latin_a", storage.Runes("a"), "0100000061", 1, false},
		{"cyrillic_ab", storage.Runes("АБ"), "020000041000000411", 2, false},
		{"cyrillic_ne", storage.Runes("не"), "020000043d00000435", 2, false},
		{"err_1_item_missed", storage.Runes{}, "05000b00210037004d", 6, true},
		{"err_all_bytes_missed", storage.Runes{}, "01", 16, true},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			wantData, err := hex.DecodeString(tt.wantW)
			require.NoError(t, err)

			buf := bytes.NewBuffer(wantData)
			newSet := make(storage.Runes, 0)
			_, readErr := newSet.ReadFrom(buf)
			require.Equalf(t, tt.wantErr, readErr != nil, "want error %v got %v", tt.wantErr, readErr)
			if readErr == nil {
				require.True(t, tt.grammemesSet.EqualTo(newSet))
			}
		})
	}
}
