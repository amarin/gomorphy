package opencorpora_test

import (
	"reflect"
	"testing"

	"github.com/amarin/binutils"

	. "github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestWordFormList_MarshalBinary(t *testing.T) {
	tests := []struct {
		name     string
		w        WordFormList
		wantData []byte
		wantErr  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData, err := tt.w.MarshalBinary()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("MarshalBinary() gotData = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestWordFormList_UnmarshalBinary(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		w       WordFormList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.w.UnmarshalBinary(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWordFormList_UnmarshalFromBuffer(t *testing.T) {
	type args struct {
		buffer *binutils.Buffer
	}
	tests := []struct {
		name    string
		w       WordFormList
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.w.UnmarshalFromBuffer(tt.args.buffer); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromBuffer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
