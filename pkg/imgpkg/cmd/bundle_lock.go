package cmd

type BundleLock struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       BundleSpec
}

type BundleSpec struct {
	Image ImageLocation
}

type ImageLock struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       ImageSpec
}

type ImageSpec struct {
	Images []ImageDesc
}

type ImageDesc struct {
	ImageLocation `yaml:",inline"`
	Name          string
	Metadata      string
}

type ImageLocation struct {
	DigestRef   string `yaml:"url,omitempty"`
	OriginalTag string `yaml:"tag,omitempty"`
}
