package envfile

import (
	"reflect"
	"testing"
)

func TestFileKeys(t *testing.T) {
	tests := []struct {
		name string
		file File
		want []string
	}{
		{"no vars", File{Path: ".env"}, []string{}},
		{
			name: "preserves file order, not sorted order",
			file: File{Path: ".env.example", Vars: []Var{
				{Key: "STRIPE_WEBHOOK_SECRET", Line: 1},
				{Key: "DATABASE_URL", Line: 2},
				{Key: "APP_ENV", Line: 3},
			}},
			want: []string{"STRIPE_WEBHOOK_SECRET", "DATABASE_URL", "APP_ENV"},
		},
		{
			name: "keeps duplicate declarations",
			file: File{Vars: []Var{{Key: "PORT", Line: 1}, {Key: "PORT", Line: 7}}},
			want: []string{"PORT", "PORT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.file.Keys()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Keys() = %v, want %v", got, tt.want)
			}
		})
	}
}
