package config

import (
	"net/url"
)

// validate checks if the Config struct is valid and returns InvalidConfigError if it's invalid.
func (c *Config) validate() error {
	var errs InvalidConfigError

	if c.MaxRetries < 0 {
		errs.appendFieldError("MaxRetries", "was %d, must be greater than or equal to 0", c.MaxRetries)
	}

	if c.Identifier == "" {
		errs.appendFieldError("Identifier", "must not be blank")
	}

	if got, limit := len(c.Identifier), 1024; got > limit {
		errs.appendFieldError("Identifier", "was %d bytes long, must not be longer than %d", got, limit)
	}

	if got, min := c.Parallelism, 1; got < min {
		errs.appendFieldError("Parallelism", "was %d, must be greater than or equal to %d", got, min)
	}

	if got, max := c.Parallelism, 1000; got > max {
		errs.appendFieldError("Parallelism", "was %d, must not be greater than %d", got, max)
	}

	if got, min := c.NodeIndex, 0; got < 0 {
		errs.appendFieldError("NodeIndex", "was %d, must be greater than or equal to %d", got, min)
	}

	if got, max := c.NodeIndex, c.Parallelism-1; got > max {
		errs.appendFieldError("NodeIndex", "was %d, must not be greater than %d", got, max)
	}

	if c.ServerBaseUrl != "" {
		if _, err := url.ParseRequestURI(c.ServerBaseUrl); err != nil {
			errs.appendFieldError("ServerBaseUrl", "must be a valid URL")
		}
	}

	if c.AccessToken == "" {
		errs.appendFieldError("AccessToken", "must not be blank")
	}

	if c.OrganizationSlug == "" {
		errs.appendFieldError("OrganizationSlug", "must not be blank")
	}

	if c.SuiteSlug == "" {
		errs.appendFieldError("SuiteSlug", "must not be blank")
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
