package conf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		typeRef interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:    "parse string",
			input:   "hello",
			typeRef: string(""),
			want:    "hello",
		},
		{
			name:    "parse int",
			input:   "123",
			typeRef: int(0),
			want:    123,
		},
		{
			name:    "parse int64",
			input:   "9223372036854775807",
			typeRef: int64(0),
			want:    int64(9223372036854775807),
		},
		{
			name:    "parse float64",
			input:   "3.14159",
			typeRef: float64(0),
			want:    3.14159,
		},
		{
			name:    "parse bool",
			input:   "true",
			typeRef: bool(false),
			want:    true,
		},
		{
			name:    "parse time",
			input:   "2024-03-20T15:04:05Z",
			typeRef: time.Time{},
			want:    time.Date(2024, 3, 20, 15, 4, 5, 0, time.UTC),
		},
		{
			name:    "parse duration",
			input:   "1h30m",
			typeRef: time.Duration(0),
			want:    90 * time.Minute,
		},
		{
			name:    "invalid int",
			input:   "not a number",
			typeRef: int(0),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.typeRef.(type) {
			case string:
				got, err := parseValue[string](tt.input)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.Equal(t, tt.want, *got)
			case int:
				got, err := parseValue[int](tt.input)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.Equal(t, tt.want, *got)
				// ... 其他类型的测试 ...
			}
		})
	}
}
