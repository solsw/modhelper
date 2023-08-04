package modhelper

import (
	"testing"
)

func TestModuleCache(t *testing.T) {
	tests := []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{name: "1",
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ModuleCache()
			if (err != nil) != tt.wantErr {
				t.Errorf("ModuleCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got != "") != tt.want {
				t.Errorf("ModuleCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
