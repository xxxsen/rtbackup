package restic

type config struct {
	pwd  string
	repo string
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
