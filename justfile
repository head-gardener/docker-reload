check:
	touch example/watchdir/file
	docker compose --file example/compose.yml down
	docker compose --file example/compose.yml up --no-start --build
	docker compose --file example/compose.yml up -d
	docker compose --file example/compose.yml logs -f &
	sleep 30
	docker compose --file example/compose.yml down
