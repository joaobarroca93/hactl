package cmd

import "testing"

// --- ensureTodoPrefix ---

func TestEnsureTodoPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"shopping_list", "todo.shopping_list"},
		{"todo.shopping_list", "todo.shopping_list"},
		{"to_do_list", "todo.to_do_list"},
		{"todo.to_do_list", "todo.to_do_list"},
	}
	for _, tt := range tests {
		got := ensureTodoPrefix(tt.input)
		if got != tt.want {
			t.Errorf("ensureTodoPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
