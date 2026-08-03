package job

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		command []string
		wantErr error
	}{
		{
			name:    "valid job",
			id:      "job1",
			command: []string{"echo", "hello"},
			wantErr: nil,
		},
		{
			name:    "missing id",
			id:      "",
			command: []string{"echo", "hello"},
			wantErr: ErrIDRequired,
		},
		{
			name:    "missing command",
			id:      "job2",
			command: nil,
			wantErr: ErrCommandRequired,
		},
		{
			name:    "empty command",
			id:      "job3",
			command: []string{},
			wantErr: ErrCommandRequired,
		},
		{
			name:    "command with empty first element",
			id:      "job4",
			command: []string{""},
			wantErr: ErrCommandRequired,
		},
		{
			name:    "command with empty first element and other elements",
			id:      "job5",
			command: []string{"", "arg1"},
			wantErr: ErrCommandRequired,
		},
		{
			name:    "valid job with multiple arguments",
			id:      "job6",
			command: []string{"echo", "hello", "world"},
			wantErr: nil,
		},
		{
			name:    "valid job with first arg but second arg empty",
			id:      "job7",
			command: []string{"echo", ""},
			wantErr: nil,
		},
		{
			name:    "empty space id",
			id:      "   ",
			command: []string{"echo", "hello"},
			wantErr: ErrIDRequired,
		},
		{
			name:    "id with leading and trailing spaces",
			id:      "  job8  ",
			command: []string{"echo", "hello"},
			wantErr: nil,
		},
		{
			name:    "blank executable",
			id:      "job9",
			command: []string{"   "},
			wantErr: ErrCommandRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j, err := New(tt.id, tt.command)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if j.ID() != strings.TrimSpace(tt.id) {
					t.Errorf("New() id = %v, want %v", j.ID(), strings.TrimSpace(tt.id))
				}
				if len(j.Command()) != len(tt.command) {
					t.Errorf("New() command length = %v, want %v", len(j.Command()), len(tt.command))
				} else {
					for i := range j.Command() {
						if j.Command()[i] != tt.command[i] {
							t.Errorf("New() command[%d] = %v, want %v", i, j.Command()[i], tt.command[i])
						}
					}
				}
				if j.Status() != StatusPending {
					t.Errorf("New() status = %v, want %v", j.Status(), StatusPending)
				}
			}

			if tt.wantErr != nil && j != nil {
				t.Errorf("New() job = %v, want nil", j)
			}
		})
	}
}

func TestNewCopiesCommand(t *testing.T) {
	originalCommand := []string{"echo", "hello"}
	j, err := New("job1", originalCommand)
	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	// Modify the original command slice
	originalCommand[0] = "modified"

	want := []string{"echo", "hello"}

	if got := j.Command(); !slices.Equal(got, want) {
		t.Errorf("Command() = %v, want %v", got, want)
	}
}

func TestJobCommandReturnsCopy(t *testing.T) {
	j, err := New("job1", []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("New() error = %v, wantErr %v", err, nil)
	}

	cmdCopy := j.Command()
	cmdCopy[0] = "modified"

	want := []string{"echo", "hello"}

	if got := j.Command(); !slices.Equal(got, want) {
		t.Errorf("Command() = %v, want %v", got, want)
	}
}
