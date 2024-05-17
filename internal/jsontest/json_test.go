package jsontest

import (
	"encoding/json"
	"testing"
)

func rawMessage(data string) *json.RawMessage {
	raw := json.RawMessage([]byte(data))
	return &raw
}

func TestJSONIsObject(t *testing.T) {
	type args struct {
		data *json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "json is object",
			args: args{data: rawMessage(`{"foo": "bar"}`)},
			want: true,
		},
		{
			name: "json is not object",
			args: args{data: rawMessage(`[{"foo": "bar}]`)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSONObject(tt.args.data); got != tt.want {
				t.Errorf("JSONIsObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONIsArray(t *testing.T) {
	type args struct {
		data *json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "json is array",
			args: args{data: rawMessage(`[{"foo": "bar}]`)},
			want: true,
		},
		{
			name: "json is not array",
			args: args{data: rawMessage(`{"foo": "bar"}`)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSONArray(tt.args.data); got != tt.want {
				t.Errorf("JSONIsArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONIsNull(t *testing.T) {
	type args struct {
		data *json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "json is null",
			args: args{data: rawMessage(`null`)},
			want: false,
		},
		{
			name: "json is not null",
			args: args{data: rawMessage(`"null"`)},
			want: false,
		},
		{
			name: "json is not null",
			args: args{data: rawMessage(``)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSONNull(tt.args.data); got != tt.want {
				t.Errorf("JSONIsNull() = %v, want %v", got, tt.want)
			}
		})
	}
}
