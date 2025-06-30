MIGRATOR=go run ./cmd/migrate
USER_FLAG=--user="$(user)"
AUTO_PUSH ?= no                       # yes => push luôn

migrate-help:
	$(MIGRATOR) help

migrate-up:
	$(MIGRATOR) up $(USER_FLAG)

migrate-down:
	$(MIGRATOR) down $(USER_FLAG)

migrate-status:
	$(MIGRATOR) status $(USER_FLAG)

migrate-version:
	$(MIGRATOR) version $(USER_FLAG)

migrate-rollback:
	$(MIGRATOR) rollback $(USER_FLAG)

migrate-create:
	$(MIGRATOR) create $(name) 

migrate-force:
	$(MIGRATOR) safe-force $(version) $(USER_FLAG)
	
migrate-sync:
	git pull
	$(MIGRATOR) up $(USER_FLAG)
migrate-commit:
	$(MIGRATOR) commit $(USER_FLAG)
migrate-push:
	@echo "▶️  Staging new migration files..."
	git add migrations/

	@# ----- Lấy file version mới nhất -----
	latest=$$(ls migrations | grep -E '^[0-9]{6}_.+\.up\.sql$$' | sort | tail -n1); \
	ver=$${latest%%_*}; \
	desc=$${latest#*_}; \
	desc=$${desc%.up.sql}; \
	msg="feat(db): $${desc} (ver $${ver#0})"; \
	echo "✅ Commit message: '$$msg'"; \
	git commit -m "$$msg"

ifeq ($(AUTO_PUSH),yes)
	@git push
endif