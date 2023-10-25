package modhelper

import (
	"reflect"
	"testing"

	"github.com/solsw/semver"
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
			args:    args{goModPath: "testdata/go.test.1.mod"},
			want:    "github.com/solsw/modhelper",
			wantErr: false,
		},
		{name: "2",
			args:    args{goModPath: "testdata/go.test.2.mod"},
			want:    "github.com/solsw/modhelper",
			wantErr: false,
		},
		{name: "3",
			args:    args{goModPath: "testdata/go.test.3.mod"},
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

func TestSemVerFromDirPath(t *testing.T) {
	type args struct {
		dirPath string
	}
	tests := []struct {
		name    string
		args    args
		want    semver.SemVer
		wantErr bool
	}{
		{name: "empty path",
			args:    args{dirPath: ""},
			wantErr: true,
		},
		{name: "no @v",
			args:    args{dirPath: "qwerty"},
			wantErr: true,
		},
		{name: "empty SemVer",
			args:    args{dirPath: "qwerty@v"},
			wantErr: true,
		},
		{name: "wrong SemVer",
			args:    args{dirPath: "qwerty@vasdfgh"},
			wantErr: true,
		},
		{name: "valid SemVer",
			args: args{dirPath: "qwerty@v1.2.3"},
			want: semver.SemVer{Major: 1, Minor: 2, Patch: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SemVerFromDirPath(tt.args.dirPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SemVerFromDirPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SemVerFromDirPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
