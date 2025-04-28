package restic

type config struct {
	pwd      string
	repo     string
	cacheDir string
}

type Option func(c *config)

func WithAuth(p string) Option {
	return func(c *config) {
		c.pwd = p
	}
}

func WithRepo(r string) Option {
	return func(c *config) {
		c.repo = r
	}
}

func WithCacheDir(d string) Option {
	return func(c *config) {
		c.cacheDir = d
	}
}
