package plugin

// Options represents the set of options that are common to all plugins.
type Options struct {
	AssetDir string
}

// Renderer defines the requirements to render manifests.
type Renderer interface {
	// Render is a method for creating the manifests required to exist in the
	// assets directory.
	Render(*Options, []string) error
}
