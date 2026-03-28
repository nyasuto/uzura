package dom

// SelectorQueryAll is a function that finds all descendant elements matching
// a CSS selector. It is set by the css package to avoid circular imports.
var SelectorQueryAll func(root Node, selector string) ([]*Element, error)

// SelectorQuery is a function that finds the first descendant element matching
// a CSS selector. It is set by the css package to avoid circular imports.
var SelectorQuery func(root Node, selector string) (*Element, error)

// SelectorMatches reports whether an element matches a CSS selector.
// It is set by the css package to avoid circular imports.
var SelectorMatches func(elem *Element, selector string) (bool, error)

// SelectorClosest finds the closest ancestor matching a CSS selector.
// It is set by the css package to avoid circular imports.
var SelectorClosest func(elem *Element, selector string) (*Element, error)
