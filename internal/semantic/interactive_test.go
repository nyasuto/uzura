package semantic

import (
	"testing"
)

func TestLinkElement(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<a href="/about">About Us</a>
		<a href="https://example.com">Example</a>
		<a>No href anchor</a>
	</body></html>`)

	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2 (anchor without href should be skipped)", len(result.Nodes))
	}
	if result.Nodes[0].Role != "link" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "link")
	}
	if result.Nodes[0].Name != "About Us" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "About Us")
	}
	if result.Nodes[0].Value != "/about" {
		t.Errorf("Value = %q, want %q", result.Nodes[0].Value, "/about")
	}
	if result.Nodes[1].Value != "https://example.com" {
		t.Errorf("Value = %q, want %q", result.Nodes[1].Value, "https://example.com")
	}
}

func TestButtonElement(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<button>Click Me</button>
		<button type="submit">Submit Form</button>
	</body></html>`)

	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(result.Nodes))
	}
	if result.Nodes[0].Role != "button" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "button")
	}
	if result.Nodes[0].Name != "Click Me" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Click Me")
	}
}

func TestInputText(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input type="text" placeholder="Enter name">
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Role != "textbox" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "textbox")
	}
	if result.Nodes[0].Name != "Enter name" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Enter name")
	}
}

func TestInputDefaultType(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input placeholder="Default text">
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Role != "textbox" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "textbox")
	}
}

func TestInputCheckbox(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input type="checkbox" name="agree">
		<input type="checkbox" name="terms" checked>
	</body></html>`)

	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(result.Nodes))
	}
	if result.Nodes[0].Role != "checkbox" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "checkbox")
	}
	if result.Nodes[0].Value != "unchecked" {
		t.Errorf("Value = %q, want %q", result.Nodes[0].Value, "unchecked")
	}
	if result.Nodes[1].Value != "checked" {
		t.Errorf("Value = %q, want %q", result.Nodes[1].Value, "checked")
	}
}

func TestInputRadio(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input type="radio" name="color" value="red">
		<input type="radio" name="color" value="blue" checked>
	</body></html>`)

	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(result.Nodes))
	}
	if result.Nodes[0].Role != "radio" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "radio")
	}
	if result.Nodes[0].Value != "unchecked" {
		t.Errorf("Value = %q, want %q", result.Nodes[0].Value, "unchecked")
	}
	if result.Nodes[1].Value != "checked" {
		t.Errorf("Value = %q, want %q", result.Nodes[1].Value, "checked")
	}
}

func TestInputSubmit(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input type="submit" value="Send">
		<input type="submit">
	</body></html>`)

	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(result.Nodes))
	}
	if result.Nodes[0].Role != "button" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "button")
	}
	if result.Nodes[0].Name != "Send" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Send")
	}
	if result.Nodes[1].Name != "Submit" {
		t.Errorf("Name = %q, want %q", result.Nodes[1].Name, "Submit")
	}
}

func TestInputHidden(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input type="hidden" name="csrf" value="token123">
	</body></html>`)

	if len(result.Nodes) != 0 {
		t.Errorf("got %d nodes, want 0 (hidden inputs should be skipped)", len(result.Nodes))
	}
}

func TestSelectElement(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<select name="country">
			<option value="jp">Japan</option>
			<option value="us" selected>United States</option>
			<option value="uk">United Kingdom</option>
		</select>
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Role != "combobox" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "combobox")
	}
	if result.Nodes[0].Value != "United States" {
		t.Errorf("Value = %q, want %q", result.Nodes[0].Value, "United States")
	}
}

func TestSelectDefaultFirst(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<select name="size">
			<option>Small</option>
			<option>Medium</option>
		</select>
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Value != "Small" {
		t.Errorf("Value = %q, want %q (should default to first option)", result.Nodes[0].Value, "Small")
	}
}

func TestTextareaElement(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<textarea placeholder="Enter comment"></textarea>
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Role != "textbox" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "textbox")
	}
	if result.Nodes[0].Name != "Enter comment" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Enter comment")
	}
}

func TestLabelForAttribute(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<label for="email">Email Address</label>
		<input type="email" id="email">
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Name != "Email Address" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Email Address")
	}
}

func TestWrappingLabel(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<label>Username <input type="text"></label>
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	if result.Nodes[0].Role != "textbox" {
		t.Errorf("Role = %q, want %q", result.Nodes[0].Role, "textbox")
	}
	// The wrapping label text includes the input's text content, which is empty
	if result.Nodes[0].Name != "Username " {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Username ")
	}
}

func TestNestedForm(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<main>
			<form>
				<label for="user">Username</label>
				<input type="text" id="user">
				<label for="pass">Password</label>
				<input type="password" id="pass">
				<input type="submit" value="Login">
			</form>
		</main>
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d top nodes, want 1 (main)", len(result.Nodes))
	}
	main := result.Nodes[0]
	if main.Role != "main" {
		t.Errorf("Role = %q, want %q", main.Role, "main")
	}
	// Children should be the 3 interactive elements (labels are not interactive)
	if len(main.Children) != 3 {
		t.Fatalf("main children = %d, want 3", len(main.Children))
	}
	if main.Children[0].Role != "textbox" {
		t.Errorf("child[0].Role = %q, want %q", main.Children[0].Role, "textbox")
	}
	if main.Children[0].Name != "Username" {
		t.Errorf("child[0].Name = %q, want %q", main.Children[0].Name, "Username")
	}
	if main.Children[1].Role != "textbox" {
		t.Errorf("child[1].Role = %q, want %q", main.Children[1].Role, "textbox")
	}
	if main.Children[2].Role != "button" {
		t.Errorf("child[2].Role = %q, want %q", main.Children[2].Role, "button")
	}
	if main.Children[2].Name != "Login" {
		t.Errorf("child[2].Name = %q, want %q", main.Children[2].Name, "Login")
	}
}

func TestAriaLabelOnInput(t *testing.T) {
	result := parseHTML(t, `<html><body>
		<input type="text" aria-label="Search query" placeholder="Search...">
	</body></html>`)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Nodes))
	}
	// aria-label should take priority over placeholder
	if result.Nodes[0].Name != "Search query" {
		t.Errorf("Name = %q, want %q", result.Nodes[0].Name, "Search query")
	}
}
