#!/bin/sh
set -eu

config_path="/app/config/config.yaml"
expect_config_value=0

for arg do
	if [ "$expect_config_value" -eq 1 ]; then
		config_path="$arg"
		expect_config_value=0
		continue
	fi

	case "$arg" in
		--config)
			expect_config_value=1
			;;
		--config=*)
			config_path="${arg#--config=}"
			;;
	esac
done

if [ ! -f "$config_path" ]; then
	cat >&2 <<EOF
Missing config file: $config_path

Create and edit the host config first:
  make config

Then run the container through Make:
  make docker-run

Or mount the config manually:
  docker run --rm \\
    -v "\$PWD/config/config.yaml:/app/config/config.yaml:ro" \\
    -v "\$PWD/output:/app/output" \\
    asana-extractor:latest
EOF
	exit 1
fi

exec asana-extractor "$@"
