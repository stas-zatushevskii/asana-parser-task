package app

import (
	"asana/internal/config"
	repository "asana/internal/repository/object"
	serviceobject "asana/internal/service/object"
	transporthttp "asana/internal/transport/http"
)

func New(configPath string) (*serviceobject.Object, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	return serviceobject.New(
		serviceobject.Config{
			Interval:  cfg.Extractor.Interval,
			PageLimit: cfg.Extractor.PageLimit,
		},
		transporthttp.NewClient(cfg.HTTP),
		repository.New(cfg.Extractor.OutputDir),
	), nil
}
