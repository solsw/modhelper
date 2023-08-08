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

func TestModulePathFromGoMod(t *testing.T) {
	type args struct {
		goModPath string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "1",
			args: args{
				goModPath: "testdata/go.test.1.mod",
			},
			want:    "github.com/solsw/modhelper",
			wantErr: false,
		},
		{name: "2",
			args: args{
				goModPath: "testdata/go.test.2.mod",
			},
			want:    "github.com/solsw/modhelper",
			wantErr: false,
		},
		{name: "3",
			args: args{
				goModPath: "testdata/go.test.3.mod",
			},
			want:    "github.com/solsw/modhelper",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ModulePathFromGoMod(tt.args.goModPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModulePathFromGoMod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("ModulePathFromGoMod() = %v, want %v", got, tt.want)
			}
		})
	}
}
