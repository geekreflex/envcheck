package check

import (
	"reflect"
	"sort"
	"testing"
)

// envcheck's central promise is that it reports variable *names* and never
// their values (see the Privacy section of the README). Problem is the type
// that crosses every output boundary — human report, JSON, and any future
// remote provider — so a value-bearing field added here would leak secrets
// everywhere at once.
//
// This test pins the field set. If you are adding a field and this fails,
// confirm it cannot carry a secret value, then update the list below
// deliberately rather than reflexively.
func TestProblemCarriesNoSecretValues(t *testing.T) {
	allowed := []string{"File", "Key", "Kind", "Line"}

	var got []string
	typ := reflect.TypeOf(Problem{})
	for i := range typ.NumField() {
		got = append(got, typ.Field(i).Name)
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, allowed) {
		t.Fatalf("check.Problem fields changed.\n got: %v\nwant: %v\n\n"+
			"Problem must never carry an environment variable's value. "+
			"If the new field is name-only metadata, add it to the allowed list.",
			got, allowed)
	}
}

func TestResultOK(t *testing.T) {
	tests := []struct {
		name string
		res  Result
		want bool
	}{
		{"zero value is clean", Result{}, true},
		{"empty slice is clean", Result{Problems: []Problem{}}, true},
		{
			name: "any problem is not clean",
			res:  Result{Problems: []Problem{{Kind: MissingInEnv, Key: "STRIPE_WEBHOOK_SECRET"}}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.res.OK(); got != tt.want {
				t.Errorf("OK() = %v, want %v", got, tt.want)
			}
		})
	}
}
